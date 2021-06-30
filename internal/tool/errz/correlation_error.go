package errz

// CorrelationError wraps an error to add a correlation id.
type CorrelationError struct {
	// Err holds the wrapped error.
	Err error
	// CorrelationId holds the correlation id.
	CorrelationId string
}

func MaybeWrapWithCorrelationId(err error, correlationId string) error {
	if err == nil || correlationId == "" {
		return err
	}
	return CorrelationError{
		Err:           err,
		CorrelationId: correlationId,
	}
}

func (e CorrelationError) Error() string {
	return e.Err.Error()
}

func (e CorrelationError) Unwrap() error {
	return e.Err
}
