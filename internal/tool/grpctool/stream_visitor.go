package grpctool

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool/automata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	eofState   protoreflect.FieldNumber = -1
	startState protoreflect.FieldNumber = -2
)

// Stream is a grpc.ServerStream or grpc.ClientStream.
type Stream interface {
	RecvMsg(m interface{}) error
}

// MessageCallback is a function with signature func(message someConcreteProtoMessage) error
// someConcreteProtoMessage must be the type passed to NewStreamVisitor().
type MessageCallback interface{}

// InvalidTransitionCallback is a callback that is called when an invalid transition is attempted.
// 'message' is nil when 'to' is eofState.
type InvalidTransitionCallback func(from, to protoreflect.FieldNumber, allowed []protoreflect.FieldNumber, message proto.Message) error

type EOFCallback func() error

// StreamVisitor allows to consume messages in a gRPC stream.
// Message order should follow the automata, defined on fields in a oneof group.
type StreamVisitor struct {
	// messageType defines the type of messages the stream consists of.
	messageType        protoreflect.MessageType
	goMessageType      reflect.Type
	allowedTransitions map[protoreflect.FieldNumber][]protoreflect.FieldNumber
	oneofDescriptor    protoreflect.OneofDescriptor
}

func NewStreamVisitor(streamMessage proto.Message) (*StreamVisitor, error) {
	messageType := streamMessage.ProtoReflect().Type()
	messageDescriptor := messageType.Descriptor()
	oneofs := messageDescriptor.Oneofs()
	l := oneofs.Len()
	if l != 1 {
		return nil, fmt.Errorf("one oneof group is expected in %s, %d defined", messageDescriptor.FullName(), l)
	}
	oneof := oneofs.Get(0)
	allowedTransitions, err := allowedTransitionsForOneof(oneof)
	if err != nil {
		return nil, err
	}
	return &StreamVisitor{
		messageType:        messageType,
		goMessageType:      reflect.TypeOf(streamMessage),
		allowedTransitions: allowedTransitions,
		oneofDescriptor:    oneof,
	}, nil
}

func (s *StreamVisitor) Visit(stream Stream, opts ...StreamVisitorOption) error {
	cfg := applyOptions(s.goMessageType, opts)
	fields := s.oneofDescriptor.Fields()
	l := fields.Len()
	for i := 0; i < l; i++ {
		field := fields.Get(i)
		fieldNumber := field.Number()
		if _, ok := cfg.callbacks[fieldNumber]; !ok {
			return fmt.Errorf("no callback defined for field %s (%d)", field.FullName(), fieldNumber)
		}
	}
	currentState := startState
	for {
		allowedTransitions := s.allowedTransitions[currentState]
		msgRefl := s.messageType.New()
		msg := msgRefl.Interface()
		err := stream.RecvMsg(msg)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			newState := eofState
			if isTransitionAllowed(newState, allowedTransitions) {
				return cfg.eofCallback()
			} else {
				return cfg.invalidTransitionCallback(currentState, newState, allowedTransitions, nil)
			}
		}
		field := msgRefl.WhichOneof(s.oneofDescriptor)
		if field == nil {
			return fmt.Errorf("no fields in the oneof group %s is set", s.oneofDescriptor.FullName())
		}
		newState := field.Number()
		if !isTransitionAllowed(newState, allowedTransitions) {
			return cfg.invalidTransitionCallback(currentState, newState, allowedTransitions, msg)
		}
		cb := cfg.callbacks[newState]
		ret := cb.Call([]reflect.Value{reflect.ValueOf(msg)})
		// It might be:
		// - an untyped nil
		// - error-typed nil
		// - non-nil error
		// Treat untyped nils as nil error since that's what it is.
		err, _ = ret[0].Interface().(error)
		if err != nil {
			return err
		}
		currentState = newState
	}
}

func allowedTransitionsForOneof(oneof protoreflect.OneofDescriptor) (map[protoreflect.FieldNumber][]protoreflect.FieldNumber, error) {
	fields := oneof.Fields()
	l := fields.Len()
	res := make(map[protoreflect.FieldNumber][]protoreflect.FieldNumber, l)
	reachable := make(map[protoreflect.FieldNumber]struct{}, l)
	for i := 0; i < l; i++ { // iterate fields of oneof
		field := fields.Get(i)
		options := field.Options().(*descriptorpb.FieldOptions)
		if !proto.HasExtension(options, automata.E_Automata) {
			return nil, fmt.Errorf("field %s does not have any transitions defined", field.FullName())
		}
		automataOption := proto.GetExtension(options, automata.E_Automata).(*automata.Automata)
		allowed, err := intsToNumbers(oneof, automataOption.NextAllowedField)
		if err != nil {
			return nil, err
		}
		res[field.Number()] = allowed
		for _, n := range allowed {
			reachable[n] = struct{}{}
		}
	}
	if len(reachable) != l {
		return nil, fmt.Errorf("not all oneof %s fields are reachable", oneof.FullName())
	}
	oneofOptions := oneof.Options().(*descriptorpb.OneofOptions)
	firstAllowedFieldsInts := proto.GetExtension(oneofOptions, automata.E_FirstAllowedField).([]int32)
	firstAllowedFieldsNumbers, err := intsToNumbers(oneof, firstAllowedFieldsInts)
	if err != nil {
		return nil, err
	}
	res[startState] = firstAllowedFieldsNumbers

	return res, nil
}

func intsToNumbers(oneofDescr protoreflect.OneofDescriptor, ints []int32) ([]protoreflect.FieldNumber, error) {
	if len(ints) == 0 {
		return nil, fmt.Errorf("empty allowed field number list in oneof %s", oneofDescr.FullName())
	}
	oneofFields := oneofDescr.Fields()
	allowed := make([]protoreflect.FieldNumber, 0, len(ints))
	for _, nextFieldInt := range ints {
		nextFieldNumber := protoreflect.FieldNumber(nextFieldInt)
		if nextFieldNumber != eofState {
			// If it's not EOF then check if it's a valid number
			nextField := oneofFields.ByNumber(nextFieldNumber)
			if nextField == nil {
				return nil, fmt.Errorf("field number %d is not part of oneof %s", nextFieldNumber, oneofDescr.FullName())
			}
		}
		allowed = append(allowed, nextFieldNumber)
	}
	return allowed, nil
}

func isTransitionAllowed(to protoreflect.FieldNumber, allowed []protoreflect.FieldNumber) bool {
	for _, n := range allowed {
		if to == n {
			return true
		}
	}
	return false
}
