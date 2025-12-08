package domain

type Driver struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	PasswordHash  string  `json:"password_hash"`
	LicenseNumber string  `json:"license_number"`
	VehicleType   string  `json:"vehicle_type"`
	VehicleAttrs  Vehicle `json:"vehicle_attrs"`
	Rating        float64 `json:"rating"`
	Status        string  `json:"status"`
	IsVerified    bool    `json:"is_verified"`
}

type CompleteRideRequest struct {
	RideID                string   `json:"ride_id"`
	FinalLocation         Location `json:"final_location"`
	ActualDistanceKm      float64  `json:"actual_distance_km"`
	ActualDurationMinutes int      `json:"actual_duration_minutes"`
}

type LocationUpdate struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters"`
	SpeedKmh       float64 `json:"speed_kmh"`
	HeadingDegrees float64 `json:"heading_degrees"`
}

type DriverOnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type DriverOfflineResponse struct {
	Status         string               `json:"status"`
	SessionID      string               `json:"session_id"`
	SessionSummary DriverSessionSummary `json:"session_summary"`
	Message        string               `json:"message"`
}

type DriverSessionSummary struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
}

type DriverCoordinateUpdate struct {
	CoordinateID string `json:"coordinate_id"`
	UpdatedAt    string `json:"updated_at"`
}
