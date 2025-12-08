package server

import (
	"errors"
	"fmt"
	"strings"
	"taxi-hailing/intenal/domain"
)

// ValidateUserInput валидирует name, email и пароль
func validateUserInput(user *domain.User, register bool) error {
	if register && len(strings.TrimSpace(user.Name)) == 0 {
		return errors.New("name cannot be empty")
	}
	if register && len(user.Name) < 2 {
		return errors.New("name too short, minimum 2 characters")
	}

	if len(strings.TrimSpace(user.Email)) == 0 {
		return errors.New("email cannot be empty")
	}
	// простая проверка email
	if !strings.Contains(user.Email, "@") || !strings.Contains(user.Email, ".") {
		return errors.New("invalid email format")
	}

	if len(user.PasswordHash) < 6 {
		return errors.New("password too short, minimum 6 characters")
	}
	if register && user.Role != "ADMIN" && user.Role != "PASSENGER" && user.Role != "DRIVER" {
		return fmt.Errorf("invalid role: %s", user.Role)
	}

	return nil
}

func validateLocation(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

func validateUpdateLocation(loc *domain.LocationUpdate) error {
	err := validateLocation(loc.Latitude, loc.Longitude)
	if err != nil {
		return err
	}
	if loc.AccuracyMeters < 0 {
		return errors.New("accuracy_meters cannot be negative")
	}
	if loc.SpeedKmh < 0 {
		return errors.New("speed_kmh cannot be negative")
	}
	if loc.HeadingDegrees < 0 || loc.HeadingDegrees >= 360 {
		return errors.New("heading_degrees must be between 0 (inclusive) and 360 (exclusive)")
	}
	return nil
}
