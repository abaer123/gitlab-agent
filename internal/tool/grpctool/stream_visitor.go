package grpctool

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"

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
	reflectMessage     protoreflect.Message
	goMessageType      reflect.Type
	allowedTransitions map[protoreflect.FieldNumber][]protoreflect.FieldNumber
	oneof              protoreflect.OneofDescriptor
}

func NewStreamVisitor(streamMessage proto.Message) (*StreamVisitor, error) {
	reflectMessage := streamMessage.ProtoReflect()
	messageDescriptor := reflectMessage.Type().Descriptor()
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
		reflectMessage:     reflectMessage,
		goMessageType:      reflect.TypeOf(streamMessage),
		allowedTransitions: allowedTransitions,
		oneof:              oneof,
	}, nil
}

func (s *StreamVisitor) Visit(stream Stream, opts ...StreamVisitorOption) error {
	cfg, err := s.applyOptions(opts)
	if err != nil {
		return err
	}
	messageType := s.reflectMessage.Type()
	currentState := startState
	for {
		allowedTransitions := s.allowedTransitions[currentState]
		msgRefl := messageType.New()
		msg := msgRefl.Interface()
		err = stream.RecvMsg(msg)
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
		field := msgRefl.WhichOneof(s.oneof)
		if field == nil {
			return fmt.Errorf("no fields in the oneof group %s is set", s.oneof.FullName())
		}
		newState := field.Number()
		if !isTransitionAllowed(newState, allowedTransitions) {
			return cfg.invalidTransitionCallback(currentState, newState, allowedTransitions, msg)
		}

		var param reflect.Value
		cb, ok := cfg.msgCallbacks[newState]
		if ok { // a message callback
			param = reflect.ValueOf(msg)
		} else { // a field callback
			cb = cfg.fieldCallbacks[newState]
			value := msgRefl.Get(field)
			switch field.Kind() { //nolint:exhaustive
			case protoreflect.MessageKind:
				param = reflect.ValueOf(value.Message().Interface())
			case protoreflect.EnumKind:
				// We have tested that values are assignable in WithCallback() so it's safe to convert here.
				param = reflect.ValueOf(value.Enum()).Convert(cb.Type().In(0))
			default:
				param = reflect.ValueOf(value.Interface())
			}
		}
		ret := cb.Call([]reflect.Value{param})

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

func (s *StreamVisitor) applyOptions(opts []StreamVisitorOption) (config, error) {
	cfg := s.defaultOptions()
	for _, o := range opts {
		err := o(&cfg)
		if err != nil {
			return config{}, err
		}
	}
	fields := s.oneof.Fields()
	l := fields.Len()
	for i := 0; i < l; i++ {
		field := fields.Get(i)
		fieldNumber := field.Number()
		_, ok := cfg.msgCallbacks[fieldNumber]
		if ok {
			continue
		}
		_, ok = cfg.fieldCallbacks[fieldNumber]
		if ok {
			continue
		}
		return config{}, fmt.Errorf("no callback defined for field %s (%d)", field.FullName(), fieldNumber)
	}
	return cfg, nil
}

func (s *StreamVisitor) defaultOptions() config {
	return config{
		reflectMessage:            s.reflectMessage,
		goMessageType:             s.goMessageType,
		oneof:                     s.oneof,
		eofCallback:               defaultEOFCallback,
		invalidTransitionCallback: defaultInvalidTransitionCallback,
		msgCallbacks:              make(map[protoreflect.FieldNumber]reflect.Value),
		fieldCallbacks:            make(map[protoreflect.FieldNumber]reflect.Value),
	}
}

func allowedTransitionsForOneof(oneof protoreflect.OneofDescriptor) (map[protoreflect.FieldNumber][]protoreflect.FieldNumber, error) {
	fields := oneof.Fields()
	l := fields.Len()
	res := make(map[protoreflect.FieldNumber][]protoreflect.FieldNumber, l)
	for i := 0; i < l; i++ { // iterate fields of oneof
		field := fields.Get(i)
		options := field.Options().(*descriptorpb.FieldOptions)
		if !proto.HasExtension(options, automata.E_NextAllowedField) {
			return nil, fmt.Errorf("field %s does not have any transitions defined", field.FullName())
		}
		nextAllowedFieldsInts := proto.GetExtension(options, automata.E_NextAllowedField).([]int32)
		nextAllowedFieldsNumbers, err := intsToNumbers(oneof, nextAllowedFieldsInts)
		if err != nil {
			return nil, err
		}
		res[field.Number()] = nextAllowedFieldsNumbers
	}
	oneofOptions := oneof.Options().(*descriptorpb.OneofOptions)
	firstAllowedFieldsInts := proto.GetExtension(oneofOptions, automata.E_FirstAllowedField).([]int32)
	firstAllowedFieldsNumbers, err := intsToNumbers(oneof, firstAllowedFieldsInts)
	if err != nil {
		return nil, err
	}
	res[startState] = firstAllowedFieldsNumbers

	unreachables := getUnreachableFields(startState, res)
	if len(unreachables) > 0 {
		return nil, fmt.Errorf("unreachable fields in oneof %s: %v", oneof.FullName(), unreachables)
	}

	return res, nil
}

func getUnreachableFields(root protoreflect.FieldNumber, graph map[protoreflect.FieldNumber][]protoreflect.FieldNumber) []protoreflect.FieldNumber {
	stack := []protoreflect.FieldNumber{root}
	remaining := make(map[protoreflect.FieldNumber]struct{}, len(graph))
	for node := range graph {
		remaining[node] = struct{}{}
	}
	delete(remaining, root)
	// non-recursive DFS, removing nodes from remaining as they are visited
	for len(stack) > 0 {
		var node protoreflect.FieldNumber
		node, stack = stack[len(stack)-1], stack[:len(stack)-1]
		for _, child := range graph[node] {
			if _, isRemaining := remaining[child]; isRemaining {
				delete(remaining, child)
				stack = append(stack, child)
			}
		}
	}
	result := make([]protoreflect.FieldNumber, 0, len(remaining))
	for f := range remaining {
		result = append(result, f)
	}
	sort.Sort(protoFieldNumbers(result)) // sort to ensure deterministic results
	return result
}

// implement sort.Interface
type protoFieldNumbers []protoreflect.FieldNumber

func (p protoFieldNumbers) Len() int           { return len(p) }
func (p protoFieldNumbers) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p protoFieldNumbers) Less(i, j int) bool { return p[i] < p[j] }

func intsToNumbers(oneof protoreflect.OneofDescriptor, ints []int32) ([]protoreflect.FieldNumber, error) {
	if len(ints) == 0 {
		return nil, fmt.Errorf("empty allowed field number list in oneof %s", oneof.FullName())
	}
	fields := oneof.Fields()
	allowed := make([]protoreflect.FieldNumber, 0, len(ints))
	for _, nextFieldInt := range ints {
		nextFieldNumber := protoreflect.FieldNumber(nextFieldInt)
		if nextFieldNumber != eofState {
			// If it's not EOF then check if it's a valid number
			nextField := fields.ByNumber(nextFieldNumber)
			if nextField == nil {
				return nil, fmt.Errorf("field number %d is not part of oneof %s", nextFieldNumber, oneof.FullName())
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
