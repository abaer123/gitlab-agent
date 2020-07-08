package protomock

import (
	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

// Eq is a better gomock.Eq() that works correctly for protobuf messages.
// Use this matcher when checking equality of structs that:
// - are v1 protobuf messages (i.e. implement "github.com/golang/protobuf/proto".Message).
// - are v2 protobuf messages (i.e. implement "google.golang.org/protobuf/proto".Message).
// - have fields of the above types.
// See https://blog.golang.org/protobuf-apiv2 for v1 vs v2 details.
func Eq(msg interface{}) gomock.Matcher {
	return protoEqMatcher{
		msg: msg,
	}
}

type protoEqMatcher struct {
	msg interface{}
}

func (e protoEqMatcher) Matches(x interface{}) bool {
	return cmp.Equal(e.msg, x, protocmp.Transform())
}

func (e protoEqMatcher) String() string {
	return fmt.Sprintf("equals %s", e.msg)
}
