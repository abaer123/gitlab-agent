package errz

import (
	"context"
	"errors"
	"io"
)

func ContextDone(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func SafeDrainAndClose(toClose io.ReadCloser, err *error) {
	if toClose == nil {
		return
	}
	const maxSlurpSize = 8 * 1024
	_, copyErr := io.CopyN(io.Discard, toClose, maxSlurpSize)
	if *err == nil {
		*err = copyErr
	}
	SafeClose(toClose, err)
}

func SafeClose(toClose io.Closer, err *error) {
	if toClose == nil {
		return
	}
	SafeCall(toClose.Close, err)
}

func SafeCall(toCall func() error, err *error) {
	e := toCall()
	if *err == nil {
		*err = e
	}
}
