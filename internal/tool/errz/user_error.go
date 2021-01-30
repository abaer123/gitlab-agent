package errz

import "fmt"

var (
	_ error = &UserError{}
)

// UserError is an error that happened because the user messed something up:
// - invalid syntax
// - invalid configuration
type UserError struct {
	// Message is a textual description of what's wrong.
	// Must be suitable to show to the user.
	Message string
	// Cause optionally holds an underlying error.
	Cause error
}

func NewUserError(msg string) error {
	return &UserError{
		Message: msg,
	}
}

func NewUserErrorf(format string, args ...interface{}) error {
	return NewUserError(fmt.Sprintf(format, args...))
}

func NewUserErrorWithCause(cause error, msg string) error {
	return &UserError{
		Message: msg,
		Cause:   cause,
	}
}

func NewUserErrorWithCausef(cause error, format string, args ...interface{}) error {
	return NewUserErrorWithCause(cause, fmt.Sprintf(format, args...))
}

func (u *UserError) Error() string {
	if u.Cause == nil {
		return u.Message
	}
	if u.Message == "" {
		return u.Cause.Error()
	}
	return fmt.Sprintf("%s: %v", u.Message, u.Cause)
}

func (u *UserError) Unwrap() error {
	return u.Cause
}
