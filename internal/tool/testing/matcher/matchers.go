package matcher

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/kube_testing"
	"google.golang.org/protobuf/testing/protocmp"
)

var (
	_ gomock.Matcher = &cmpMatcher{}
	_ gomock.Matcher = &errorMatcher{}
)

// ProtoEq is a better gomock.Eq() that works correctly for protobuf messages.
// Use this matcher when checking equality of structs that:
// - are v1 protobuf messages (i.e. implement "github.com/golang/protobuf/proto".Message).
// - are v2 protobuf messages (i.e. implement "google.golang.org/protobuf/proto".Message).
// - have fields of the above types.
// See https://blog.golang.org/protobuf-apiv2 for v1 vs v2 details.
func ProtoEq(t *testing.T, msg interface{}) gomock.Matcher {
	return Cmp(t, msg, protocmp.Transform())
}

func ErrorEq(expectedError string) gomock.Matcher {
	return &errorMatcher{
		expectedError: expectedError,
	}
}

func K8sObjectEq(t *testing.T, obj interface{}, opts ...cmp.Option) gomock.Matcher {
	o := []cmp.Option{kube_testing.TransformToUnstructured(), cmpopts.EquateEmpty()}
	o = append(o, opts...)
	return Cmp(t, obj, o...)
}

func Cmp(t *testing.T, expected interface{}, opts ...cmp.Option) gomock.Matcher {
	return &cmpMatcher{
		t:        t,
		expected: expected,
		options:  opts,
	}
}

type cmpMatcher struct {
	t        *testing.T
	expected interface{}
	options  []cmp.Option
}

func (e cmpMatcher) Matches(x interface{}) bool {
	equal := cmp.Equal(e.expected, x, e.options...)
	if !equal && e.t != nil {
		e.t.Log(cmp.Diff(e.expected, x, e.options...))
	}
	return equal
}

func (e cmpMatcher) String() string {
	return fmt.Sprintf("equals %s with %d option(s)", e.expected, len(e.options))
}

type errorMatcher struct {
	expectedError string
}

func (e *errorMatcher) Matches(x interface{}) bool {
	if err, ok := x.(error); ok {
		return err.Error() == e.expectedError
	}
	return false
}

func (e *errorMatcher) String() string {
	return fmt.Sprintf("error with message %q", e.expectedError)
}
