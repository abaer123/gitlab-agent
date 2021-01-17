package grpctool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool/test"
)

func TestRawCodecRoundtrip(t *testing.T) {
	input := &RawFrame{Data: []byte{1, 2, 3, 4, 5}}
	serialized, err := RawCodec{}.Marshal(input)
	require.NoError(t, err)
	output := &RawFrame{}
	err = RawCodec{}.Unmarshal(serialized, output)
	require.NoError(t, err)
	assert.Equal(t, input, output)
}

func TestRawCodecBadType(t *testing.T) {
	serialized, err := RawCodec{}.Marshal(&test.Request{})
	require.EqualError(t, err, "RawCodec.Marshal(): unexpected source message type: *test.Request")
	assert.Empty(t, serialized)

	output := &test.Request{}
	err = RawCodec{}.Unmarshal([]byte{1, 2, 3, 4, 5}, output)
	require.EqualError(t, err, "RawCodec.Unmarshal(): unexpected target message type: *test.Request")
}
