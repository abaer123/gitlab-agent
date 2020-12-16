package errz

import (
	"context"
	"errors"
	"io"
)

func ContextDone(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
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
