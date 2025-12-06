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
