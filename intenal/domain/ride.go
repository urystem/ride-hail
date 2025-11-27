package domain

import (
	"time"
)

type User struct {
	ID           string         `db:"id" json:"id"`
	Name         string         `db:"name" json:"name"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
	Email        string         `db:"email" json:"email"`
	Role         string         `db:"role" json:"role"`
	Status       string         `db:"status" json:"status"`
	PasswordHash string         `db:"password_hash" json:"password"` // не экспортировать в JSON
	Attrs        map[string]any `db:"attrs" json:"attrs"`
}

// for sql
type RideRequest struct {
	PassengerID          string  `json:"passenger_id"`
	PickupLatitude       float64 `json:"pickup_latitude"`
	PickupLongitude      float64 `json:"pickup_longitude"`
	PickupAddress        string  `json:"pickup_address"`
	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	DestinationAddress   string  `json:"destination_address"`
	Priority             uint
	EstimatedFare        float64
	FinalFare            float64
	RideType             string `json:"ride_type"`
}

type RideResponse struct {
	RideID                   string  `json:"ride_id"`
	RideNumber               string  `json:"ride_number"`
	Status                   string  `json:"status"`
	EstimatedFare            float64 `json:"estimated_fare"`
	EstimatedDurationMinutes int     `json:"estimated_duration_minutes"`
	EstimatedDistanceKM      float64 `json:"estimated_distance_km"`
}

type CancelRideRequest struct {
	Reason string `json:"reason"`
}

type CancelRideResponse struct {
	RideID      string    `json:"ride_id"`
	Status      string    `json:"status"`
	CancelledAt time.Time `json:"cancelled_at"`
	Message     string    `json:"message"`
}

type LocationCoordinateUpdate struct {
	DriverID string `json:"driver_id"`
	RideID   string `json:"ride_id"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	FareAmount     float64
	Distance       float64
	DurationMinute time.Duration
	OldCoorID      string
}

// rabbit
type RideStatusUpdate struct {
	RideID        string    `json:"ride_id"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	DriverID      string    `json:"driver_id"`
	CorrelationID string    `json:"correlation_id"`
}

type Coordinates struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

type RideRequestRabbit struct {
	RideID              string      `json:"ride_id"`
	RideNumber          string      `json:"ride_number"`
	PickupLocation      Coordinates `json:"pickup_location"`
	DestinationLocation Coordinates `json:"destination_location"`
	RideType            string      `json:"ride_type"`
	EstimatedFare       float64     `json:"estimated_fare"`
	MaxDistanceKM       float64     `json:"max_distance_km"`
	TimeoutSeconds      int         `json:"timeout_seconds"`
	CorrelationID       string      `json:"correlation_id"`
}
