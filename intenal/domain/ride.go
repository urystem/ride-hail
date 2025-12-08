package domain

import "time"

// kerek
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

// http
type RideResponse struct {
	RideID                   string  `json:"ride_id"`
	RideNumber               string  `json:"ride_number"`
	Status                   string  `json:"status"`
	EstimatedFare            float64 `json:"estimated_fare"`
	EstimatedDurationMinutes int     `json:"estimated_duration_minutes"`
	EstimatedDistanceKM      float64 `json:"estimated_distance_km"`
	BaseFare                 float64 //for sql coordinate
}

// http
type CancelRideRequest struct {
	Reason string `json:"reason"`
}

// http
type CancelRideResponse struct {
	RideID      string    `json:"ride_id"`
	Status      string    `json:"status"`
	CancelledAt time.Time `json:"cancelled_at"`
	Message     string    `json:"message"`
}

// belgisiz
type LocationCoordinateUpdate struct {
	DriverID string `json:"driver_id"`
	RideID   string `json:"ride_id"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	FareAmount     float64
	Distance       float64
	DurationMinute int
}

// rabbit
type RideStatusUpdate struct {
	RideID        string    `json:"ride_id"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	DriverID      string    `json:"driver_id"`
	CorrelationID string    `json:"correlation_id"`
}

// rabbit kerek
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

// driver income
type RideResponseMatch struct {
	RideID              string     `json:"ride_id"`
	DriverID            string     `json:"driver_id"`
	Accepted            bool       `json:"accepted"`
	EstimatedArrivalMin int        `json:"estimated_arrival_minutes"`
	DriverLocation      Location   `json:"driver_location"`
	DriverInfo          DriverInfo `json:"driver_info"`
	CorrelationID       string     `json:"correlation_id"`
	EstimatedArrival    time.Time  `json:"estimated_arrival"`
}

// driver income
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// driver income
type DriverInfo struct {
	Name    string  `json:"name"`
	Rating  float64 `json:"rating"`
	Vehicle Vehicle `json:"vehicle"`
}

// driver income
type Vehicle struct {
	Make        string `json:"make"`
	Model       string `json:"model"`
	Color       string `json:"color"`
	Plate       string `json:"plate"`
	VehicleYear uint16 `json:"vehicle_year"`
}

// ws
type RideStatusUpdateMatched struct {
	Type          string       `json:"type"`
	RideID        string       `json:"ride_id"`
	RideNumber    string       `json:"ride_number"`
	Status        string       `json:"status"`
	DriverInfo    DriverInfoWs `json:"driver_info"`
	CorrelationID string       `json:"correlation_id"`
}

// ws
type DriverInfoWs struct {
	DriverID string `json:"driver_id"`
	DriverInfo
}

type DriverLocationUpdate struct {
	DriverID string `json:"driver_id"`
	RideID   string `json:"ride_id"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	SpeedKmh       float64   `json:"speed_kmh"`
	HeadingDegrees float64   `json:"heading_degrees"`
	Timestamp      time.Time `json:"timestamp"`
}

type CoordinateUpdate struct {
	UpdatedAt       time.Time `db:"updated_at"`
	Latitude        float64   `db:"latitude"`
	Longitude       float64   `db:"longitude"`
	FareAmount      float64   `db:"fare_amount"` // если NULL не будет, то просто float64
	DistanceKm      float64   `db:"distance_km"`
	DurationMinutes int       `db:"duration_minutes"`
}
