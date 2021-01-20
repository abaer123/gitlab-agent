package grpctool

import (
	"fmt"

	"google.golang.org/grpc/encoding/proto"
)

// RawCodec is a *raw* encoding.Codec.
// This codec treats a gRPC message frame as raw bytes.
type RawCodec struct {
}

func (c RawCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.(*RawFrame)
	if !ok {
		return nil, fmt.Errorf("RawCodec.Marshal(): unexpected source message type: %T", v)
	}
	return out.Data, nil
}

func (c RawCodec) Unmarshal(data []byte, v interface{}) error {
	dst, ok := v.(*RawFrame)
	if !ok {
		return fmt.Errorf("RawCodec.Unmarshal(): unexpected target message type: %T", v)
	}
	dst.Data = data
	return nil
}

func (c RawCodec) Name() string {
	// Pretend to be a codec for protobuf.
	return proto.Name
}

// String is here for compatibility with grpc.Codec interface.
func (c RawCodec) String() string {
	return c.Name()
}

type RawFrame struct {
	Data []byte
}
