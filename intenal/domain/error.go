package domain

import "errors"

var (
	ErrNotFound = errors.New("not found")
	Errconflict = errors.New("conflict")
	
)
