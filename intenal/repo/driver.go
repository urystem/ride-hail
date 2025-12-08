package repo

import (
	"context"
	"fmt"
	"taxi-hailing/intenal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DriverRepo struct {
	db *pgxpool.Pool
}

func NewDriverRepo(db *pgxpool.Pool) *DriverRepo {
	return &DriverRepo{db: db}
}

func (r *DriverRepo) CreateDriver(ctx context.Context, driver *domain.Driver) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, status)
		VALUES ($1, $2, $3, 'DRIVER','ACTIVE')
		RETURNING id
	`, driver.Name, driver.Email, driver.PasswordHash).Scan(&driver.ID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO drivers (
			id, license_number, vehicle_type, vehicle_attrs, is_verified
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`, driver.ID, driver.LicenseNumber, driver.VehicleType, driver.VehicleAttrs, driver.IsVerified)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpdateDriverStatus changes the driver's status (e.g., "AVAILABLE", "BUSY", "EN_ROUTE", "OFFLINE") and inserts into driver_sessions when status goes to AVAILABLE.
// If transitioning to AVAILABLE, create a new session (driver went online).
func (r *DriverRepo) UpdateDriverToOnline(ctx context.Context, driverID uuid.UUID, location *domain.Location) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	// Get current status before updating
	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM drivers WHERE id=$1`, driverID).Scan(&currentStatus)
	if err != nil {
		return uuid.Nil, err
	}
	if currentStatus != "OFFLINE" {
		return uuid.Nil, fmt.Errorf("driver is not offline")
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'AVAILABLE', updated_at = now()
		WHERE id = $2
	`, driverID)
	if err != nil {
		return uuid.Nil, err
	}

	var sessionID uuid.UUID
	err = tx.QueryRow(ctx, `
			INSERT INTO driver_sessions (driver_id)
			VALUES ($1)
			RETURNING id
		`, driverID).Scan(&sessionID)
	if err != nil {
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (driver_id, latitude, longitude)
		VALUES ($1, $2, $3)
	`, driverID, location.Lat, location.Lng)
	if err != nil {
		return uuid.Nil, err
	}
	return sessionID, tx.Commit(ctx)
}

func (r *DriverRepo) UpdateDriverToOffline(ctx context.Context, driverID uuid.UUID) (*uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM drivers WHERE id=$1`, driverID).Scan(&currentStatus)
	if err != nil {
		return nil, err
	}

	if currentStatus != "AVAILABLE" {
		return nil, fmt.Errorf("driver is not available")
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'OFFLINE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return nil, err
	}

	var sessionID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE driver_sessions
		SET ended_at = now()
		WHERE driver_id = $1
		AND ended_at IS NULL
		RETURNING id
	`, driverID).Scan(&sessionID)
	if err != nil {
		return nil, err
	}

	return &sessionID, tx.Commit(ctx)
}

func (r *DriverRepo) GetDriverSessionSummary(ctx context.Context, sessionID *uuid.UUID) (*domain.DriverSessionSummary, error) {
	if sessionID == nil {
		return nil, fmt.Errorf("sessionID is nil")
	}

	summary := new(domain.DriverSessionSummary)
	err := r.db.QueryRow(ctx, `
		SELECT 
			COALESCE(EXTRACT(EPOCH FROM (ended_at - started_at))/3600, 0) AS duration_hours,
			COALESCE(total_rides, 0),
			COALESCE(total_earnings, 0)
		FROM driver_sessions
		WHERE id = $1 AND ended_at IS NOT NULL
	`, sessionID).Scan(&summary.DurationHours, &summary.RidesCompleted, &summary.Earnings)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func (r *DriverRepo) UpdateDriverToEnRoute(ctx context.Context, driverID uuid.UUID, req *domain.DriverLocationMessage) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM drivers WHERE id=$1`, driverID).Scan(&currentStatus)
	if err != nil {
		return err
	}

	if currentStatus != "AVAILABLE" {
		return fmt.Errorf("driver is not busy")
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'EN_ROUTE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return err
	}

	var pickupCoordinateID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT pickup_coordinate_id FROM rides WHERE id = $1
	`, req.RideID).Scan(&pickupCoordinateID)
	if err != nil {
		return fmt.Errorf("cannot get pickup coordinate id for ride: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, ride_id)
		VALUES ($1, $2, $3, $4, $5)
	`, pickupCoordinateID, driverID, req.DriverLocation.Latitude, req.DriverLocation.Longitude, req.RideID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *DriverRepo) UpdateDriverToBusy(ctx context.Context, driverID uuid.UUID, req *domain.DriverLocationMessage) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM drivers WHERE id=$1`, driverID).Scan(&currentStatus)
	if err != nil {
		return err
	}
	if currentStatus != "EN_ROUTE" {
		return fmt.Errorf("driver is not EN_ROUTE")
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'BUSY', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return err
	}

	var destinationCoordinateID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT destination_coordinate_id FROM rides WHERE id = $1
	`, req.RideID).Scan(&destinationCoordinateID)
	if err != nil {
		return fmt.Errorf("cannot get destination coordinate id for ride: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, ride_id)
		VALUES ($1, $2, $3, $4)
	`, destinationCoordinateID, driverID, req.DriverLocation.Latitude, req.DriverLocation.Longitude, req.RideID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DriverRepo) CompleteRide(ctx context.Context, driverID uuid.UUID, req *domain.CompleteRideRequest) (float64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM drivers WHERE id=$1`, driverID).Scan(&currentStatus)
	if err != nil {
		return 0, err
	}
	if currentStatus != "BUSY" {
		return 0, fmt.Errorf("driver is not BUSY")
	}
	// update driver status to AVAILABLE, updated_at
	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'AVAILABLE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return 0, fmt.Errorf("cannot update driver: %w", err)
	}

	// Validate driver/ride relationship, must be driver of this ride & status BUSY
	var dbDriverID uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `
		SELECT driver_id, status FROM rides WHERE id = $1
	`, req.RideID).Scan(&dbDriverID, &status)
	if err != nil {
		return 0, fmt.Errorf("cannot load ride: %w", err)
	}
	if dbDriverID != driverID {
		return 0, fmt.Errorf("driver is not assigned to this ride")
	}
	if status != "IN_PROGRESS" {
		return 0, fmt.Errorf("ride is not in progress")
	}

	var destinationCoordinateID, passengerID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT passenger_id, destination_coordinate_id FROM rides WHERE id = $1
	`, req.RideID).Scan(&destinationCoordinateID, &passengerID)
	if err != nil {
		return 0, fmt.Errorf("cannot get destination coordinate id for ride: %w", err)
	}

	// Write to location_history
	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, recorded_at, ride_id)
		VALUES ($1, $2, $3, $4, now(), $5)
	`, destinationCoordinateID, driverID, req.FinalLocation.Lat, req.FinalLocation.Lng, req.RideID)
	if err != nil {
		return 0, fmt.Errorf("cannot insert location_history: %w", err)
	}

	// Increment driver_sessions for current session: add ride and earnings
	var fare float64
	err = tx.QueryRow(ctx, `
    SELECT fare_amount
    FROM coordinates
    WHERE entity_id = $1 AND is_current = TRUE`, passengerID).Scan(&fare)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE driver_sessions
		SET 
			total_rides = total_rides + 1,
			total_earnings = total_earnings+ $1
		WHERE driver_id = $2
			AND ended_at IS NULL
	`, fare, driverID)

	if err != nil {
		return 0, fmt.Errorf("cannot update driver_sessions: %w", err)
	}

	return fare, tx.Commit(ctx)
}

// UpdateDriverLocation updates or inserts a driver's latest location.
// Если драйвер по rideId куда-то едет (то есть у него статус EN_ROUTE или BUSY),
// то location_history связывается с соответствующей поездкой.
// Ошибку не выдаем, если драйвер OFFLINE — просто записываем его координаты.

func (r *DriverRepo) UpdateDriverLocation(ctx context.Context, driverID uuid.UUID, loc *domain.LocationUpdate) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	// Проверим статус драйвера, если надо (для rideID)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM drivers WHERE id=$1`, driverID).Scan(&status)
	if err != nil {
		return uuid.Nil, err
	}
	var returnID uuid.UUID
	switch status {
	case "OFFLINE":
		return uuid.Nil, fmt.Errorf("cannot update driver offines")
	case "AVAILABLE":
		err = tx.QueryRow(ctx, `
		INSERT INTO location_history (
			driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`,
			driverID,
			loc.Latitude,
			loc.Longitude,
			loc.AccuracyMeters,
			loc.SpeedKmh,
			loc.HeadingDegrees,
		).Scan(&returnID)
		if err != nil {
			return uuid.Nil, err
		}
	case "BUSY", "EN_ROUTE":
		var rideID, coordinateID uuid.UUID
		err = tx.QueryRow(ctx, `
			SELECT ride_id, coordinate_id
			FROM location_history 
			WHERE driver_id = $1 
			ORDER BY recorded_at DESC 
			LIMIT 1
		`, driverID).Scan(&rideID, &coordinateID)

		if err != nil {
			return uuid.Nil, fmt.Errorf("cannot get rideID from location_history for driver: %w", err)
		}
		err = tx.QueryRow(ctx, `
		INSERT INTO location_history (
			coordinate_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees, ride_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
			coordinateID,
			driverID,
			loc.Latitude,
			loc.Longitude,
			loc.AccuracyMeters,
			loc.SpeedKmh,
			loc.HeadingDegrees,
			rideID,
		).Scan(&returnID)

		if err != nil {
			return uuid.Nil, err
		}
	default:
		return uuid.Nil, fmt.Errorf("invalid driver status: %s", status)
	}
	return returnID, tx.Commit(ctx)
}
