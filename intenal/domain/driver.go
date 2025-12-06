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
	RideID               string    `json:"ride_id"`
	FinalLocation        Location  `json:"final_location"`
	ActualDistanceKm     float64   `json:"actual_distance_km"`
	ActualDurationMinutes int      `json:"actual_duration_minutes"`
}

