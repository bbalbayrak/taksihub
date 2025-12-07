package repository

import "errors"

var (
	ErrDriverNotFound      = errors.New("driver not found")
	ErrDriverAlreadyExists = errors.New("driver already exists")
	ErrInvalidID           = errors.New("invalid driver ID")
	ErrInvalidCoordinates  = errors.New("invalid coordinates")
	ErrInvalidRadius       = errors.New("invalid radius")
	ErrDatabaseError       = errors.New("database error")
)
