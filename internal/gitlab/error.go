package gitlab

import (
	"errors"
	"fmt"
)

type ErrorKind int

const (
	ErrorKindOther ErrorKind = iota
	ErrorKindForbidden
	ErrorKindUnauthorized
)

type ClientError struct {
	Kind       ErrorKind
	StatusCode int
}

func (c *ClientError) Error() string {
	return fmt.Sprintf("error kind: %d; status: %d", c.Kind, c.StatusCode)
}

func IsForbidden(err error) bool {
	var e *ClientError
	if !errors.As(err, &e) {
		return false
	}
	return e.Kind == ErrorKindForbidden
}

func IsUnauthorized(err error) bool {
	var e *ClientError
	if !errors.As(err, &e) {
		return false
	}
	return e.Kind == ErrorKindUnauthorized
}

func IsClientError(err error) bool {
	var e *ClientError
	if !errors.As(err, &e) {
		return false
	}
	return e.StatusCode >= 400 && e.StatusCode < 500
}

func IsServerError(err error) bool {
	var e *ClientError
	if !errors.As(err, &e) {
		return false
	}
	return e.StatusCode >= 500 && e.StatusCode < 600
}
