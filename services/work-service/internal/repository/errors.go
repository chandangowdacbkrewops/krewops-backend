package repository

import "errors"

var (
	ErrDuplicateApplication       = errors.New("application already exists")
	ErrApplicationAlreadyAccepted = errors.New("no worker seats remaining for this work")
	ErrDuplicateAssignment        = errors.New("assignment already exists")
)
