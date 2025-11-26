package repo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"taxi-hailing/intenal/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ride struct {
	slogg *slog.Logger
	*pgxpool.Pool
}

func NewRideRepo(slogg *slog.Logger, pool *pgxpool.Pool) any {
	return &ride{
		slogg: slogg,
		Pool:  pool,
	}
}

func (p *ride) RegisterPassenger(ctx context.Context, user *domain.User) (string, error) {
	var id string
	err := p.QueryRow(ctx, `
			INSERT INTO users (name, email, role, password_hash)
			VALUES ($1, $2, 'PASSENGER', $3)
			RETURNING id`,
		user.Name, user.Email, user.PasswordHash).Scan(&id)
	return id, err
}

func (p *ride) GetPassword(ctx context.Context, email string) (string, error) {
	var passwordHash string
	err := p.QueryRow(ctx, `
		SELECT password_hash
		FROM users
		WHERE email = $1
	`, email).Scan(&passwordHash)
	return passwordHash, err
}

func (p *ride) CreateRideTx(ctx context.Context, r *domain.RideRequest, res *domain.RideResponse) error {
	// Начинаем транзакцию
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var pickupID, destID, rideID string

	// Вставляем pickup координату
	err = tx.QueryRow(ctx, `
        INSERT INTO coordinates (
        entity_id,
        entity_type,
        address,
        latitude,
        longitude,
        fare_amount,
        distance_km,
        duration_minutes)
        VALUES ($1, 'passenger', $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, r.PassengerID,
		r.PickupAddress,
		r.PickupLatitude,
		r.PickupLongitude,
		res.EstimatedFare,
		res.EstimatedDistanceKM,
		res.EstimatedDurationMinutes,
	).Scan(&pickupID)
	if err != nil {
		return err
	}

	// Вставляем destination координату
	err = tx.QueryRow(ctx, `
        INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
        VALUES ($1, 'passenger', $2, $3, $4, false)
        RETURNING id
    `, r.PassengerID, r.DestinationAddress, r.DestinationLatitude, r.DestinationLongitude).Scan(&destID)
	if err != nil {
		return err
	}

	// Считаем, сколько поездок уже было сегодня
	var count int
	err = tx.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM rides
        WHERE created_at::date = CURRENT_DATE
    `).Scan(&count)
	if err != nil {
		return err
	}

	// Создаём поездку
	rideNumber := fmt.Sprintf("RIDE_%s_%03d", time.Now().Format("20060102"), count+1) // упрощённо
	err = tx.QueryRow(ctx, `
        INSERT INTO rides (ride_number, passenger_id, vehicle_type, status, priority, pickup_coordinate_id, destination_coordinate_id, estimated_fare)
        VALUES ($1, $2, $3, 'REQUESTED',$4, $5, $6)
        RETURNING id
    `, rideNumber, r.PassengerID, r.RideType, r.Priority, pickupID, destID, res.EstimatedFare).Scan(&rideID)
	if err != nil {
		return err
	}
	res.RideID = rideID
	res.RideNumber = rideNumber
	_, err = tx.Exec(ctx, `
    INSERT INTO ride_events (ride_id, event_type, event_data)
    VALUES ($1, 'RIDE_REQUESTED', $2::jsonb)
`, rideID, res)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *ride) CancelRide(ctx context.Context, rideID string, stu *domain.CancelRideRequest) error {
	// Начинаем транзакцию
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	err = tx.QueryRow(ctx, `
        UPDATE rides
        SET status = 'CANCELLED',
            cancelled_at = now(),
            cancellation_reason = $2,
            updated_at = now()
        WHERE id = $1
        RETURNING OLD.status
    `, rideID, stu.Reason).Scan(&oldStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound // или своя ошибка
		}
		return err
	}
	// Формируем event_data
	eventData := map[string]any{
		"reason":     stu.Reason,
		"old_status": oldStatus,
		"new_status": "CANCELLED",
	}
	_, err = tx.Exec(ctx, `
        INSERT INTO ride_events (ride_id, event_type, event_data)
        VALUES ($1, 'RIDE_CANCELLED', $2::jsonb)
    `, rideID, eventData)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *ride) RideMatchedUpdate(ctx context.Context, data *domain.DriverLocationUpdate) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var oldStatus string
	err = tx.QueryRow(ctx, `
        SELECT status
        FROM rides
        WHERE id = $1
    `, data.RideID).Scan(&oldStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound // или своя ошибка
		}
		return err
	}
	if oldStatus != "REQUESTED" {
		return fmt.Errorf("invalid old status: %s", oldStatus)
	}
	_, err = tx.Exec(ctx, `
        UPDATE rides
        SET
            status = 'MATCHED',
            driver_id= $2,
            matched_at = now(),
            updated_at = now()
        WHERE id = $1`, data.RideID, data.DriverID)
	if err != nil {
		return err
	}

	// 2️⃣ Формируем данные события
	eventData := map[string]any{
		"old_status": oldStatus,
		"new_status": "MATCHED",
		"driver_id":  data.DriverID,
		"location": map[string]float64{
			"lat": data.Location.Lat,
			"lng": data.Location.Lng,
		},
		// "estimated_arrival": data.EstimatedArrival, // time.Time или строка в ISO8601
	}

	// 3️⃣ Вставляем событие в ride_events
	_, err = tx.Exec(ctx, `
        INSERT INTO ride_events (ride_id, event_type, event_data)
        VALUES ($1, 'DRIVER_MATCHED', $2::jsonb)
    `, data.RideID, eventData)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *ride) RideEnRouteUpdate(ctx context.Context, data *domain.DriverLocationUpdate) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	err = tx.QueryRow(ctx, `
        SELECT status
        FROM rides
        WHERE id = $1
    `, data.RideID).Scan(&oldStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound // или своя ошибка
		}
		return err
	}
	if oldStatus != "MATCHED" {
		return fmt.Errorf("invalid old status: %s", oldStatus)
	}

	var driverID string

	err = tx.QueryRow(ctx, `
    SELECT driver_id::text
    FROM rides
    WHERE id = $1
`, data.RideID).Scan(&driverID)
	if err != nil {
		return err
	}
	if driverID != data.DriverID {
		return fmt.Errorf("invalid driver id: %s != %s", data.DriverID, driverID)
	}

	_, err = tx.Exec(ctx, `
        UPDATE rides
        SET
            status = 'EN_ROUTE',
            updated_at = now()
        WHERE id = $1`, data.RideID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *ride) RideArrivedUpdate(ctx context.Context, data *domain.DriverLocationUpdate) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var oldStatus string
	err = tx.QueryRow(ctx, `
        SELECT status
        FROM rides
        WHERE id = $1`, data.RideID).Scan(&oldStatus)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound // или своя ошибка
		}
		return err
	}

	if oldStatus != "EN_ROUTE" {
		return fmt.Errorf("invalid old status: %s", oldStatus)
	}

	// 	var driverID string

	// 	err = tx.QueryRow(ctx, `
	//     SELECT driver_id::text
	//     FROM rides
	//     WHERE id = $1
	// `, data.RideID).Scan(&driverID)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if driverID != data.DriverID {
	// 		return fmt.Errorf("invalid driver id: %s != %s", data.DriverID, driverID)
	// 	}
	_, err = tx.Exec(ctx, `
        UPDATE rides
        SET
            status = 'ARRIVED',
            updated_at = now(),
            arrived_at = now()
        WHERE id = $1`, data.RideID)
	if err != nil {
		return err
	}

	eventData := map[string]any{
		"old_status": oldStatus,
		"new_status": "DRIVER_ARRIVED",
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO ride_events (ride_id, event_type, event_data)
        VALUES ($1, 'DRIVER_ARRIVED', $2::jsonb)
    `, data.RideID, eventData)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *ride) RideInProgressUpdate(ctx context.Context, data *domain.DriverLocationUpdate) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var oldStatus string
	err = tx.QueryRow(ctx, `
        SELECT status
        FROM rides
        WHERE id = $1`, data.RideID).Scan(&oldStatus)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound // или своя ошибка
		}
		return err
	}

	if oldStatus != "ARRIVED" {
		return fmt.Errorf("invalid old status: %s", oldStatus)
	}

	var driverID string
	err = tx.QueryRow(ctx, `
    SELECT driver_id::text
    FROM rides
    WHERE id = $1
`, data.RideID).Scan(&driverID)
	if err != nil {
		return err
	}
	if driverID != data.DriverID {
		return fmt.Errorf("invalid driver id: %s != %s", data.DriverID, driverID)
	}

	_, err = tx.Exec(ctx, `
        UPDATE rides
        SET
            status = 'IN_PROGRESS',
            updated_at = now(),
            started_at = now()
        WHERE id = $1`, data.RideID)
	if err != nil {
		return err
	}
	eventData := map[string]any{
		"old_status": oldStatus,
		"new_status": "RIDE_STARTED",
		"driver_id":  data.DriverID,
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO ride_events (ride_id, event_type, event_data)
        VALUES ($1, 'RIDE_STARTED', $2::jsonb)
    `, data.RideID, eventData)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)

}

func (p *ride) RideCompleteUpdate(ctx context.Context, data *domain.RideStatusUpdate) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var oldStatus, passengerID string
	err = tx.QueryRow(ctx, `
        SELECT status, passenger_id
        FROM rides
        WHERE id = $1`, data.RideID).Scan(&oldStatus, &passengerID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound // или своя ошибка
		}
		return err
	}

	if oldStatus != "IN_PROGRESS" {
		return fmt.Errorf("invalid old status: %s", oldStatus)
	}
	// 	var driverID string
	// 	err = tx.QueryRow(ctx, `
	//     SELECT driver_id::text
	//     FROM rides
	//     WHERE id = $1
	// `, data.RideID).Scan(&driverID)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if driverID != data.DriverID {
	// 		return fmt.Errorf("invalid driver id: %s != %s", data.DriverID, driverID)
	// 	}

	var fare float64

	err = tx.QueryRow(ctx, `
    SELECT fare_amount
    FROM coordinates
    WHERE entity_id = $1 AND is_current = TRUE`, passengerID).Scan(&fare)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
        UPDATE rides
        SET
            status = 'COMPLETED',
            updated_at = now(),
            completed_at = now(),
			final_fare = $2,
        WHERE id = $1`, data.RideID, fare)
	if err != nil {
		return err
	}
	eventData := map[string]any{
		"old_status": oldStatus,
		"new_status": "COMPLETED",
		"driver_id":  data.DriverID,
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO ride_events (ride_id, event_type, event_data)
        VALUES ($1, 'RIDE_STARTED', $2::jsonb)
    `, data.RideID, eventData)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *ride) RideLocationUpdate(ctx context.Context, data *domain.LocationCoordinateUpdate) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var oldStatus, drID, coorID, passengerID string
	err = tx.QueryRow(ctx, `
        SELECT status, driver_id::text, pickup_coordinate_id, passenger_id
        FROM rides
        WHERE id = $1`, data.RideID).Scan(&oldStatus, &drID, &coorID, &passengerID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound // или своя ошибка
		}
		return err
	}
	if oldStatus != "IN_PROGRESS" {
		return fmt.Errorf("invalid status : %s", oldStatus)
	}

	if drID != data.DriverID {
		return fmt.Errorf("invalid driver id: %s != %s", drID, data.DriverID)
	}

	var pickupID string
	err = tx.QueryRow(ctx, `
        INSERT INTO coordinates (
        entity_id,
        entity_type,
        address,
        latitude,
        longitude,
        fare_amount,
        distance_km,
        duration_minutes)
        VALUES ($1, 'passenger', $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, passengerID,
		data.Location,
		data.Location.Lat,
		data.Location.Lng,
		data.FareAmount,
		data.Distance,
		data.DurationMinute,
	).Scan(&pickupID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
        UPDATE coordinates
        SET 
            is_current = false,
            updated_at = NOW()
        WHERE id = $1
    `, data.OldCoorID)
	if err != nil {
		return err
	}

	var esminatedFare float64
	err = tx.QueryRow(ctx, `
        UPDATE rides
        SET
            final_fare = $2,
            updated_at = now()
        WHERE id = $1
		RETURNING estimated_fare;
    `, data.RideID, data.FareAmount).Scan(&esminatedFare)
	if err != nil {
		return err
	}

	if esminatedFare < data.FareAmount {
		eventData := map[string]any{
			"raznicha": data.FareAmount - esminatedFare,
		}

		_, err = tx.Exec(ctx, `
        INSERT INTO ride_events (ride_id, event_type, event_data)
        VALUES ($1, 'FARE_ADJUSTED', $2::jsonb)
    `, data.RideID, eventData)
		if err != nil {
			return err
		}
	}

	eventData := map[string]any{
		"location_updated": data.Location,
	}
	_, err = tx.Exec(ctx, `
        INSERT INTO ride_events (ride_id, event_type, event_data)
        VALUES ($1, 'LOCATION_UPDATED', $2::jsonb)
    `, data.RideID, eventData)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

//get passenger info for driver
