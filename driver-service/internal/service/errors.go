package service

import (
	"errors"
)

var (
	ErrDriverNotFound      = errors.New("driver not found")
	ErrDriverAlreadyExists = errors.New("driver already exists")
	ErrInvalidID           = errors.New("invalid driver ID")
	ErrInvalidPlate        = errors.New("invalid license plate")
	ErrInvalidLocation     = errors.New("invalid location coordinates")
	ErrInvalidTaxiType     = errors.New("invalid taxi type")
	ErrValidationFailed    = errors.New("validation failed")
	ErrRepositoryError     = errors.New("repository error")
)
