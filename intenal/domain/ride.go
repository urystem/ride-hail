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
	PasswordHash string         `db:"password_hash" json:"-"` // не экспортировать в JSON
	Attrs        map[string]any `db:"attrs" json:"attrs"`
}

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

type DriverLocationUpdate struct {
	DriverID string `json:"driver_id"`
	RideID   string `json:"ride_id"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	SpeedKMH       float64   `json:"speed_kmh"`
	HeadingDegrees float64   `json:"heading_degrees"`
	Timestamp      time.Time `json:"timestamp"`
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

type RideStatusUpdate struct {
	DriverID string `json:"driver_id"`
	RideID   string `json:"ride_id"`
}

type RideStatusCompleteUpdate struct{
	
}