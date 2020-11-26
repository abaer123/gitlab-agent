package protodefault

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestNotNil(t *testing.T) {
	var x struct {
		Bla *durationpb.Duration // an example proto field
	}
	NotNil(&x.Bla)
	assert.NotNil(t, x.Bla)
}
