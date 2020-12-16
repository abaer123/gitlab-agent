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
	reflectMessage            protoreflect.Message
	goMessageType             reflect.Type
	oneof                     protoreflect.OneofDescriptor
	eofCallback               EOFCallback
	invalidTransitionCallback InvalidTransitionCallback
	msgCallbacks              map[protoreflect.FieldNumber]reflect.Value // callbacks that accept the whole message
	fieldCallbacks            map[protoreflect.FieldNumber]reflect.Value // callbacks that accept a specific field type of the oneof
}

type StreamVisitorOption func(*config) error

func WithEOFCallback(cb EOFCallback) StreamVisitorOption {
	return func(c *config) error {
		c.eofCallback = cb
		return nil
	}
}

// WithCallback registers cb to be called when entering transitionTo when parsing the stream. Only one callback can be registered per target
func WithCallback(transitionTo protoreflect.FieldNumber, cb MessageCallback) StreamVisitorOption {
	cbType := reflect.TypeOf(cb)
	if cbType.Kind() != reflect.Func {
		panic(fmt.Errorf("cb must be a function, got: %T", cb))
	}
	if cbType.NumIn() != 1 {
		panic(fmt.Errorf("cb must take one parameter only, got: %T", cb))
	}
	if cbType.NumOut() != 1 {
		panic(fmt.Errorf("cb must return one value only, got: %T", cb))
	}
	if cbType.Out(0) != errorType {
		panic(fmt.Errorf("cb must return an error, got: %T", cb))
	}
	return func(c *config) error {
		if existingCb, exists := c.msgCallbacks[transitionTo]; exists {
			return fmt.Errorf("callback for %d has already been defined: %v", transitionTo, existingCb)
		}
		if existingCb, exists := c.fieldCallbacks[transitionTo]; exists {
			return fmt.Errorf("callback for %d has already been defined: %v", transitionTo, existingCb)
		}
		field := c.oneof.Fields().ByNumber(transitionTo)
		if field == nil {
			return fmt.Errorf("oneof %s does not have a field %d", c.oneof.FullName(), transitionTo)
		}
		inType := cbType.In(0)
		if c.goMessageType.AssignableTo(inType) {
			c.msgCallbacks[transitionTo] = reflect.ValueOf(cb)
		} else if reflect.TypeOf(c.reflectMessage.Get(field).Message().Interface()).AssignableTo(inType) {
			c.fieldCallbacks[transitionTo] = reflect.ValueOf(cb)
		} else {
			return fmt.Errorf("callback must be a function with one parameter of type %s or one of the oneof field types, got: %T", c.goMessageType, cb)
		}
		return nil
	}
}

func WithInvalidTransitionCallback(cb InvalidTransitionCallback) StreamVisitorOption {
	return func(c *config) error {
		c.invalidTransitionCallback = cb
		return nil
	}
}

func defaultInvalidTransitionCallback(from, to protoreflect.FieldNumber, allowed []protoreflect.FieldNumber, message proto.Message) error {
	return fmt.Errorf("transition from %d to %d is not allowed. Allowed: %d", from, to, allowed)
}

func defaultEOFCallback() error {
	return nil
}
