package db

import "errors"

// ErrNotFound is returned when a document is not found in the database
var ErrNotFound = errors.New("document not found")
