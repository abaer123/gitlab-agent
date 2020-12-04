package grpctool

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)

type config struct {
	goMessageType             reflect.Type
	eofCallback               EOFCallback
	invalidTransitionCallback InvalidTransitionCallback
	callbacks                 map[protoreflect.FieldNumber]reflect.Value
}

type StreamVisitorOption func(*config)

func applyOptions(goMessageType reflect.Type, opts []StreamVisitorOption) config {
	cfg := defaultOptions(goMessageType)
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

func defaultOptions(goMessageType reflect.Type) config {
	return config{
		goMessageType:             goMessageType,
		eofCallback:               defaultEOFCallback,
		invalidTransitionCallback: defaultInvalidTransitionCallback,
		callbacks:                 make(map[protoreflect.FieldNumber]reflect.Value),
	}
}

func WithEOFCallback(cb EOFCallback) StreamVisitorOption {
	return func(c *config) {
		c.eofCallback = cb
	}
}

// WithCallback registers cb to be called when entering transitionTo when parsing the stream. Only one callback can be registered per target
func WithCallback(transitionTo protoreflect.FieldNumber, cb MessageCallback) StreamVisitorOption {
	t := reflect.TypeOf(cb)
	if t.Kind() != reflect.Func {
		panic(fmt.Errorf("cb must be a function, got: %T", cb))
	}
	if t.NumIn() != 1 {
		panic(fmt.Errorf("cb must take one parameter only, got: %T", cb))
	}
	if t.NumOut() != 1 {
		panic(fmt.Errorf("cb must return one value only, got: %T", cb))
	}
	if t.Out(0) != errorType {
		panic(fmt.Errorf("cb must return an error, got: %T", cb))
	}
	return func(c *config) {
		if t.In(0) != c.goMessageType {
			panic(fmt.Errorf("callback must be a function with one parameter of type %s, got: %T", c.goMessageType, cb))
		}
		if _, exists := c.callbacks[transitionTo]; exists {
			panic(fmt.Errorf("callback for %d has already been defined", transitionTo))
		}
		c.callbacks[transitionTo] = reflect.ValueOf(cb)
	}
}

func WithInvalidTransitionCallback(cb InvalidTransitionCallback) StreamVisitorOption {
	return func(c *config) {
		c.invalidTransitionCallback = cb
	}
}

func defaultInvalidTransitionCallback(from, to protoreflect.FieldNumber, allowed []protoreflect.FieldNumber, message proto.Message) error {
	return fmt.Errorf("transition from %d to %d is not allowed. Allowed: %d", from, to, allowed)
}

func defaultEOFCallback() error {
	return nil
}
