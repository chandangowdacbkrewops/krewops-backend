package repository

import "errors"

var (
	ErrDuplicateApplication       = errors.New("application already exists")
	ErrApplicationAlreadyAccepted = errors.New("another application has already been accepted")
)
