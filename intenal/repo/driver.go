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

func (r *DriverRepo) UpdateDriverToOffline(ctx context.Context, driverID uuid.UUID) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM drivers WHERE id=$1`, driverID).Scan(&currentStatus)
	if err != nil {
		return uuid.Nil, err
	}

	if currentStatus != "AVAILABLE" {
		return uuid.Nil, fmt.Errorf("driver is not available")
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'OFFLINE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return uuid.Nil, err
	}

	var sessionID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE driver_sessions
		SET ended_at = now()
		WHERE driver_id = $1
		AND ended_at IS NULL
	`, driverID).Scan(&sessionID)
	if err != nil {
		return uuid.Nil, err
	}
	return sessionID, tx.Commit(ctx)
}

func (r *DriverRepo) UpdateDriverToEnRoute(ctx context.Context, driverID uuid.UUID, rideID uuid.UUID, location *domain.Location) error {
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
	`, rideID).Scan(&pickupCoordinateID)
	if err != nil {
		return fmt.Errorf("cannot get pickup coordinate id for ride: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, ride_id)
		VALUES ($1, $2, $3, $4, $5)
	`, pickupCoordinateID, driverID, location.Lat, location.Lng, rideID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *DriverRepo) UpdateDriverToBusy(ctx context.Context, driverID uuid.UUID, rideID uuid.UUID, location *domain.Location) error {
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
	`, rideID).Scan(&destinationCoordinateID)
	if err != nil {
		return fmt.Errorf("cannot get destination coordinate id for ride: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, ride_id)
		VALUES ($1, $2, $3, $4)
	`, destinationCoordinateID, driverID, location.Lat, location.Lng, rideID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DriverRepo) CompleteRide(ctx context.Context, driverID uuid.UUID, req *domain.CompleteRideRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Validate driver/ride relationship, must be driver of this ride & status BUSY
	var dbDriverID uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `
		SELECT driver_id, status FROM rides WHERE id = $1
	`, req.RideID).Scan(&dbDriverID, &status)
	if err != nil {
		return fmt.Errorf("cannot load ride: %w", err)
	}
	if dbDriverID != driverID {
		return fmt.Errorf("driver is not assigned to this ride")
	}
	if status != "IN_PROGRESS" && status != "BUSY" {
		return fmt.Errorf("ride is not in progress")
	}

	// update ride: set status, completed_at, final stats
	_, err = tx.Exec(ctx, `
		UPDATE rides
		SET
			status = 'COMPLETED',
			completed_at = now(),
			final_fare = COALESCE(final_fare, estimated_fare),
			updated_at = now(),
			distance_km = $1,
			duration_minutes = $2
		WHERE id = $3
	`, req.ActualDistanceKm, req.ActualDurationMinutes, req.RideID)
	if err != nil {
		return fmt.Errorf("cannot update ride data: %w", err)
	}

	// update driver status to AVAILABLE, updated_at
	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'AVAILABLE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return fmt.Errorf("cannot update driver: %w", err)
	}

	// Mark the final arrival location as not current for previous locations
	_, err = tx.Exec(ctx, `
		UPDATE coordinates
		SET is_current = false, updated_at = now()
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
	`, driverID)
	if err != nil {
		return fmt.Errorf("cannot update coordinates: %w", err)
	}
	// Insert new coordinates for driver's final dropoff
	var coordID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO coordinates (
			entity_id,
			entity_type,
			address,
			latitude,
			longitude,
			is_current
		) VALUES ($1, 'driver', '', $2, $3, true)
		RETURNING id
	`, driverID, req.FinalLocation.Lat, req.FinalLocation.Lng).Scan(&coordID)
	if err != nil {
		return fmt.Errorf("cannot insert coordinate: %w", err)
	}

	// Write to location_history
	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, recorded_at, ride_id)
		VALUES ($1, $2, $3, $4, now(), $5)
	`, coordID, driverID, req.FinalLocation.Lat, req.FinalLocation.Lng, req.RideID)
	if err != nil {
		return fmt.Errorf("cannot insert location_history: %w", err)
	}

	// Increment driver's total_rides and total_earnings
	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET total_rides = total_rides + 1, total_earnings = total_earnings + COALESCE(
			(SELECT final_fare FROM rides WHERE id = $1), 0), updated_at = now()
		WHERE id = $2
	`, req.RideID, driverID)
	if err != nil {
		return fmt.Errorf("cannot update driver stats: %w", err)
	}

	return tx.Commit(ctx)
}
