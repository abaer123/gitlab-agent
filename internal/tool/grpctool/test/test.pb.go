// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.13.0
// source: internal/tool/grpctool/test/test.proto

package test

import (
	reflect "reflect"
	sync "sync"

	proto "github.com/golang/protobuf/proto"
	_ "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool/automata"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type Response struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Message:
	//	*Response_First_
	//	*Response_Data_
	//	*Response_Last_
	Message isResponse_Message `protobuf_oneof:"message"`
}

func (x *Response) Reset() {
	*x = Response{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response) ProtoMessage() {}

func (x *Response) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response.ProtoReflect.Descriptor instead.
func (*Response) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{0}
}

func (m *Response) GetMessage() isResponse_Message {
	if m != nil {
		return m.Message
	}
	return nil
}

func (x *Response) GetFirst() *Response_First {
	if x, ok := x.GetMessage().(*Response_First_); ok {
		return x.First
	}
	return nil
}

func (x *Response) GetData() *Response_Data {
	if x, ok := x.GetMessage().(*Response_Data_); ok {
		return x.Data
	}
	return nil
}

func (x *Response) GetLast() *Response_Last {
	if x, ok := x.GetMessage().(*Response_Last_); ok {
		return x.Last
	}
	return nil
}

type isResponse_Message interface {
	isResponse_Message()
}

type Response_First_ struct {
	First *Response_First `protobuf:"bytes,1,opt,name=first,proto3,oneof"`
}

type Response_Data_ struct {
	Data *Response_Data `protobuf:"bytes,2,opt,name=data,proto3,oneof"`
}

type Response_Last_ struct {
	Last *Response_Last `protobuf:"bytes,3,opt,name=last,proto3,oneof"`
}

func (*Response_First_) isResponse_Message() {}

func (*Response_Data_) isResponse_Message() {}

func (*Response_Last_) isResponse_Message() {}

type NoOneofs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *NoOneofs) Reset() {
	*x = NoOneofs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NoOneofs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NoOneofs) ProtoMessage() {}

func (x *NoOneofs) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NoOneofs.ProtoReflect.Descriptor instead.
func (*NoOneofs) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{1}
}

type TwoOneofs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Message1:
	//	*TwoOneofs_M11
	//	*TwoOneofs_M12
	Message1 isTwoOneofs_Message1 `protobuf_oneof:"message1"`
	// Types that are assignable to Message2:
	//	*TwoOneofs_M21
	//	*TwoOneofs_M22
	Message2 isTwoOneofs_Message2 `protobuf_oneof:"message2"`
}

func (x *TwoOneofs) Reset() {
	*x = TwoOneofs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TwoOneofs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TwoOneofs) ProtoMessage() {}

func (x *TwoOneofs) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TwoOneofs.ProtoReflect.Descriptor instead.
func (*TwoOneofs) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{2}
}

func (m *TwoOneofs) GetMessage1() isTwoOneofs_Message1 {
	if m != nil {
		return m.Message1
	}
	return nil
}

func (x *TwoOneofs) GetM11() int32 {
	if x, ok := x.GetMessage1().(*TwoOneofs_M11); ok {
		return x.M11
	}
	return 0
}

func (x *TwoOneofs) GetM12() int32 {
	if x, ok := x.GetMessage1().(*TwoOneofs_M12); ok {
		return x.M12
	}
	return 0
}

func (m *TwoOneofs) GetMessage2() isTwoOneofs_Message2 {
	if m != nil {
		return m.Message2
	}
	return nil
}

func (x *TwoOneofs) GetM21() int32 {
	if x, ok := x.GetMessage2().(*TwoOneofs_M21); ok {
		return x.M21
	}
	return 0
}

func (x *TwoOneofs) GetM22() int32 {
	if x, ok := x.GetMessage2().(*TwoOneofs_M22); ok {
		return x.M22
	}
	return 0
}

type isTwoOneofs_Message1 interface {
	isTwoOneofs_Message1()
}

type TwoOneofs_M11 struct {
	M11 int32 `protobuf:"varint,1,opt,name=m11,proto3,oneof"`
}

type TwoOneofs_M12 struct {
	M12 int32 `protobuf:"varint,2,opt,name=m12,proto3,oneof"`
}

func (*TwoOneofs_M11) isTwoOneofs_Message1() {}

func (*TwoOneofs_M12) isTwoOneofs_Message1() {}

type isTwoOneofs_Message2 interface {
	isTwoOneofs_Message2()
}

type TwoOneofs_M21 struct {
	M21 int32 `protobuf:"varint,3,opt,name=m21,proto3,oneof"`
}

type TwoOneofs_M22 struct {
	M22 int32 `protobuf:"varint,4,opt,name=m22,proto3,oneof"`
}

func (*TwoOneofs_M21) isTwoOneofs_Message2() {}

func (*TwoOneofs_M22) isTwoOneofs_Message2() {}

type TwoValidOneofs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Message1:
	//	*TwoValidOneofs_M11
	//	*TwoValidOneofs_M12
	Message1 isTwoValidOneofs_Message1 `protobuf_oneof:"message1"`
	// Types that are assignable to Message2:
	//	*TwoValidOneofs_M21
	//	*TwoValidOneofs_M22
	Message2 isTwoValidOneofs_Message2 `protobuf_oneof:"message2"`
}

func (x *TwoValidOneofs) Reset() {
	*x = TwoValidOneofs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TwoValidOneofs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TwoValidOneofs) ProtoMessage() {}

func (x *TwoValidOneofs) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TwoValidOneofs.ProtoReflect.Descriptor instead.
func (*TwoValidOneofs) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{3}
}

func (m *TwoValidOneofs) GetMessage1() isTwoValidOneofs_Message1 {
	if m != nil {
		return m.Message1
	}
	return nil
}

func (x *TwoValidOneofs) GetM11() int32 {
	if x, ok := x.GetMessage1().(*TwoValidOneofs_M11); ok {
		return x.M11
	}
	return 0
}

func (x *TwoValidOneofs) GetM12() int32 {
	if x, ok := x.GetMessage1().(*TwoValidOneofs_M12); ok {
		return x.M12
	}
	return 0
}

func (m *TwoValidOneofs) GetMessage2() isTwoValidOneofs_Message2 {
	if m != nil {
		return m.Message2
	}
	return nil
}

func (x *TwoValidOneofs) GetM21() int32 {
	if x, ok := x.GetMessage2().(*TwoValidOneofs_M21); ok {
		return x.M21
	}
	return 0
}

func (x *TwoValidOneofs) GetM22() int32 {
	if x, ok := x.GetMessage2().(*TwoValidOneofs_M22); ok {
		return x.M22
	}
	return 0
}

type isTwoValidOneofs_Message1 interface {
	isTwoValidOneofs_Message1()
}

type TwoValidOneofs_M11 struct {
	M11 int32 `protobuf:"varint,1,opt,name=m11,proto3,oneof"`
}

type TwoValidOneofs_M12 struct {
	M12 int32 `protobuf:"varint,2,opt,name=m12,proto3,oneof"`
}

func (*TwoValidOneofs_M11) isTwoValidOneofs_Message1() {}

func (*TwoValidOneofs_M12) isTwoValidOneofs_Message1() {}

type isTwoValidOneofs_Message2 interface {
	isTwoValidOneofs_Message2()
}

type TwoValidOneofs_M21 struct {
	M21 int32 `protobuf:"varint,3,opt,name=m21,proto3,oneof"`
}

type TwoValidOneofs_M22 struct {
	M22 int32 `protobuf:"varint,4,opt,name=m22,proto3,oneof"`
}

func (*TwoValidOneofs_M21) isTwoValidOneofs_Message2() {}

func (*TwoValidOneofs_M22) isTwoValidOneofs_Message2() {}

type OutOfOneof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	X int32 `protobuf:"varint,1,opt,name=x,proto3" json:"x,omitempty"`
	// Types that are assignable to Message:
	//	*OutOfOneof_M1
	//	*OutOfOneof_M2
	Message isOutOfOneof_Message `protobuf_oneof:"message"`
}

func (x *OutOfOneof) Reset() {
	*x = OutOfOneof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OutOfOneof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OutOfOneof) ProtoMessage() {}

func (x *OutOfOneof) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OutOfOneof.ProtoReflect.Descriptor instead.
func (*OutOfOneof) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{4}
}

func (x *OutOfOneof) GetX() int32 {
	if x != nil {
		return x.X
	}
	return 0
}

func (m *OutOfOneof) GetMessage() isOutOfOneof_Message {
	if m != nil {
		return m.Message
	}
	return nil
}

func (x *OutOfOneof) GetM1() int32 {
	if x, ok := x.GetMessage().(*OutOfOneof_M1); ok {
		return x.M1
	}
	return 0
}

func (x *OutOfOneof) GetM2() int32 {
	if x, ok := x.GetMessage().(*OutOfOneof_M2); ok {
		return x.M2
	}
	return 0
}

type isOutOfOneof_Message interface {
	isOutOfOneof_Message()
}

type OutOfOneof_M1 struct {
	M1 int32 `protobuf:"varint,2,opt,name=m1,proto3,oneof"`
}

type OutOfOneof_M2 struct {
	M2 int32 `protobuf:"varint,3,opt,name=m2,proto3,oneof"`
}

func (*OutOfOneof_M1) isOutOfOneof_Message() {}

func (*OutOfOneof_M2) isOutOfOneof_Message() {}

type NotAllReachable struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Message:
	//	*NotAllReachable_M1
	//	*NotAllReachable_M2
	//	*NotAllReachable_M3
	Message isNotAllReachable_Message `protobuf_oneof:"message"`
}

func (x *NotAllReachable) Reset() {
	*x = NotAllReachable{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotAllReachable) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotAllReachable) ProtoMessage() {}

func (x *NotAllReachable) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotAllReachable.ProtoReflect.Descriptor instead.
func (*NotAllReachable) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{5}
}

func (m *NotAllReachable) GetMessage() isNotAllReachable_Message {
	if m != nil {
		return m.Message
	}
	return nil
}

func (x *NotAllReachable) GetM1() int32 {
	if x, ok := x.GetMessage().(*NotAllReachable_M1); ok {
		return x.M1
	}
	return 0
}

func (x *NotAllReachable) GetM2() int32 {
	if x, ok := x.GetMessage().(*NotAllReachable_M2); ok {
		return x.M2
	}
	return 0
}

func (x *NotAllReachable) GetM3() int32 {
	if x, ok := x.GetMessage().(*NotAllReachable_M3); ok {
		return x.M3
	}
	return 0
}

type isNotAllReachable_Message interface {
	isNotAllReachable_Message()
}

type NotAllReachable_M1 struct {
	M1 int32 `protobuf:"varint,1,opt,name=m1,proto3,oneof"`
}

type NotAllReachable_M2 struct {
	M2 int32 `protobuf:"varint,2,opt,name=m2,proto3,oneof"`
}

type NotAllReachable_M3 struct {
	M3 int32 `protobuf:"varint,3,opt,name=m3,proto3,oneof"`
}

func (*NotAllReachable_M1) isNotAllReachable_Message() {}

func (*NotAllReachable_M2) isNotAllReachable_Message() {}

func (*NotAllReachable_M3) isNotAllReachable_Message() {}

type Response_First struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Response_First) Reset() {
	*x = Response_First{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response_First) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response_First) ProtoMessage() {}

func (x *Response_First) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response_First.ProtoReflect.Descriptor instead.
func (*Response_First) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{0, 0}
}

type Response_Data struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Response_Data) Reset() {
	*x = Response_Data{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response_Data) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response_Data) ProtoMessage() {}

func (x *Response_Data) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response_Data.ProtoReflect.Descriptor instead.
func (*Response_Data) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{0, 1}
}

func (x *Response_Data) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type Response_Last struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Response_Last) Reset() {
	*x = Response_Last{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response_Last) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response_Last) ProtoMessage() {}

func (x *Response_Last) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_test_test_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response_Last.ProtoReflect.Descriptor instead.
func (*Response_Last) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_test_test_proto_rawDescGZIP(), []int{0, 2}
}

var File_internal_tool_grpctool_test_test_proto protoreflect.FileDescriptor

var file_internal_tool_grpctool_test_test_proto_rawDesc = []byte{
	0x0a, 0x26, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x6f, 0x6f, 0x6c, 0x2f,
	0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x74, 0x65,
	0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x74, 0x65, 0x73, 0x74, 0x1a, 0x2e,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x67, 0x72,
	0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x61, 0x75, 0x74, 0x6f, 0x6d, 0x61, 0x74, 0x61, 0x2f,
	0x61, 0x75, 0x74, 0x6f, 0x6d, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xf3,
	0x01, 0x0a, 0x08, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x34, 0x0a, 0x05, 0x66,
	0x69, 0x72, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x74, 0x65, 0x73,
	0x74, 0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x46, 0x69, 0x72, 0x73, 0x74,
	0x42, 0x06, 0x82, 0xf6, 0x2c, 0x02, 0x08, 0x02, 0x48, 0x00, 0x52, 0x05, 0x66, 0x69, 0x72, 0x73,
	0x74, 0x12, 0x37, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x13, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e,
	0x44, 0x61, 0x74, 0x61, 0x42, 0x0c, 0x82, 0xf6, 0x2c, 0x02, 0x08, 0x02, 0x82, 0xf6, 0x2c, 0x02,
	0x08, 0x03, 0x48, 0x00, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x3a, 0x0a, 0x04, 0x6c, 0x61,
	0x73, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x4c, 0x61, 0x73, 0x74, 0x42, 0x0f, 0x82,
	0xf6, 0x2c, 0x0b, 0x08, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x48, 0x00,
	0x52, 0x04, 0x6c, 0x61, 0x73, 0x74, 0x1a, 0x07, 0x0a, 0x05, 0x46, 0x69, 0x72, 0x73, 0x74, 0x1a,
	0x1a, 0x0a, 0x04, 0x44, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x1a, 0x06, 0x0a, 0x04, 0x4c,
	0x61, 0x73, 0x74, 0x42, 0x0f, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x04,
	0x88, 0xf6, 0x2c, 0x01, 0x22, 0x0a, 0x0a, 0x08, 0x4e, 0x6f, 0x4f, 0x6e, 0x65, 0x6f, 0x66, 0x73,
	0x22, 0x7f, 0x0a, 0x09, 0x54, 0x77, 0x6f, 0x4f, 0x6e, 0x65, 0x6f, 0x66, 0x73, 0x12, 0x12, 0x0a,
	0x03, 0x6d, 0x31, 0x31, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x48, 0x00, 0x52, 0x03, 0x6d, 0x31,
	0x31, 0x12, 0x12, 0x0a, 0x03, 0x6d, 0x31, 0x32, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x48, 0x00,
	0x52, 0x03, 0x6d, 0x31, 0x32, 0x12, 0x12, 0x0a, 0x03, 0x6d, 0x32, 0x31, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x05, 0x48, 0x01, 0x52, 0x03, 0x6d, 0x32, 0x31, 0x12, 0x12, 0x0a, 0x03, 0x6d, 0x32, 0x32,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x48, 0x01, 0x52, 0x03, 0x6d, 0x32, 0x32, 0x42, 0x10, 0x0a,
	0x08, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x31, 0x12, 0x04, 0x88, 0xf6, 0x2c, 0x01, 0x42,
	0x10, 0x0a, 0x08, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0x12, 0x04, 0x88, 0xf6, 0x2c,
	0x03, 0x22, 0xb6, 0x01, 0x0a, 0x0e, 0x54, 0x77, 0x6f, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x4f, 0x6e,
	0x65, 0x6f, 0x66, 0x73, 0x12, 0x1a, 0x0a, 0x03, 0x6d, 0x31, 0x31, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x05, 0x42, 0x06, 0x82, 0xf6, 0x2c, 0x02, 0x08, 0x02, 0x48, 0x00, 0x52, 0x03, 0x6d, 0x31, 0x31,
	0x12, 0x23, 0x0a, 0x03, 0x6d, 0x31, 0x32, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x42, 0x0f, 0x82,
	0xf6, 0x2c, 0x0b, 0x08, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x48, 0x00,
	0x52, 0x03, 0x6d, 0x31, 0x32, 0x12, 0x1a, 0x0a, 0x03, 0x6d, 0x32, 0x31, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x05, 0x42, 0x06, 0x82, 0xf6, 0x2c, 0x02, 0x08, 0x04, 0x48, 0x01, 0x52, 0x03, 0x6d, 0x32,
	0x31, 0x12, 0x23, 0x0a, 0x03, 0x6d, 0x32, 0x32, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x42, 0x0f,
	0x82, 0xf6, 0x2c, 0x0b, 0x08, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x48,
	0x01, 0x52, 0x03, 0x6d, 0x32, 0x32, 0x42, 0x10, 0x0a, 0x08, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x31, 0x12, 0x04, 0x88, 0xf6, 0x2c, 0x01, 0x42, 0x10, 0x0a, 0x08, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x32, 0x12, 0x04, 0x88, 0xf6, 0x2c, 0x03, 0x22, 0x68, 0x0a, 0x0a, 0x4f, 0x75,
	0x74, 0x4f, 0x66, 0x4f, 0x6e, 0x65, 0x6f, 0x66, 0x12, 0x0c, 0x0a, 0x01, 0x78, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x01, 0x78, 0x12, 0x18, 0x0a, 0x02, 0x6d, 0x31, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x05, 0x42, 0x06, 0x82, 0xf6, 0x2c, 0x02, 0x08, 0x01, 0x48, 0x00, 0x52, 0x02, 0x6d, 0x31,
	0x12, 0x21, 0x0a, 0x02, 0x6d, 0x32, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x42, 0x0f, 0x82, 0xf6,
	0x2c, 0x0b, 0x08, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x48, 0x00, 0x52,
	0x02, 0x6d, 0x32, 0x42, 0x0f, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x04,
	0x88, 0xf6, 0x2c, 0x02, 0x22, 0x79, 0x0a, 0x0f, 0x4e, 0x6f, 0x74, 0x41, 0x6c, 0x6c, 0x52, 0x65,
	0x61, 0x63, 0x68, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x18, 0x0a, 0x02, 0x6d, 0x31, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x42, 0x06, 0x82, 0xf6, 0x2c, 0x02, 0x08, 0x02, 0x48, 0x00, 0x52, 0x02, 0x6d,
	0x31, 0x12, 0x18, 0x0a, 0x02, 0x6d, 0x32, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x42, 0x06, 0x82,
	0xf6, 0x2c, 0x02, 0x08, 0x01, 0x48, 0x00, 0x52, 0x02, 0x6d, 0x32, 0x12, 0x21, 0x0a, 0x02, 0x6d,
	0x33, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x42, 0x0f, 0x82, 0xf6, 0x2c, 0x0b, 0x08, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x48, 0x00, 0x52, 0x02, 0x6d, 0x33, 0x42, 0x0f,
	0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x04, 0x88, 0xf6, 0x2c, 0x03, 0x42,
	0x5d, 0x5a, 0x5b, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x69,
	0x74, 0x6c, 0x61, 0x62, 0x2d, 0x6f, 0x72, 0x67, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x2d, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x67, 0x69, 0x74,
	0x6c, 0x61, 0x62, 0x2d, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e,
	0x61, 0x6c, 0x2f, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c,
	0x2f, 0x61, 0x75, 0x74, 0x6f, 0x6d, 0x61, 0x74, 0x61, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_tool_grpctool_test_test_proto_rawDescOnce sync.Once
	file_internal_tool_grpctool_test_test_proto_rawDescData = file_internal_tool_grpctool_test_test_proto_rawDesc
)

func file_internal_tool_grpctool_test_test_proto_rawDescGZIP() []byte {
	file_internal_tool_grpctool_test_test_proto_rawDescOnce.Do(func() {
		file_internal_tool_grpctool_test_test_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_tool_grpctool_test_test_proto_rawDescData)
	})
	return file_internal_tool_grpctool_test_test_proto_rawDescData
}

var file_internal_tool_grpctool_test_test_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_internal_tool_grpctool_test_test_proto_goTypes = []interface{}{
	(*Response)(nil),        // 0: test.Response
	(*NoOneofs)(nil),        // 1: test.NoOneofs
	(*TwoOneofs)(nil),       // 2: test.TwoOneofs
	(*TwoValidOneofs)(nil),  // 3: test.TwoValidOneofs
	(*OutOfOneof)(nil),      // 4: test.OutOfOneof
	(*NotAllReachable)(nil), // 5: test.NotAllReachable
	(*Response_First)(nil),  // 6: test.Response.First
	(*Response_Data)(nil),   // 7: test.Response.Data
	(*Response_Last)(nil),   // 8: test.Response.Last
}
var file_internal_tool_grpctool_test_test_proto_depIdxs = []int32{
	6, // 0: test.Response.first:type_name -> test.Response.First
	7, // 1: test.Response.data:type_name -> test.Response.Data
	8, // 2: test.Response.last:type_name -> test.Response.Last
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_internal_tool_grpctool_test_test_proto_init() }
func file_internal_tool_grpctool_test_test_proto_init() {
	if File_internal_tool_grpctool_test_test_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_tool_grpctool_test_test_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_tool_grpctool_test_test_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NoOneofs); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_tool_grpctool_test_test_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TwoOneofs); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_tool_grpctool_test_test_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TwoValidOneofs); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_tool_grpctool_test_test_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OutOfOneof); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_tool_grpctool_test_test_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NotAllReachable); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_tool_grpctool_test_test_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response_First); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_tool_grpctool_test_test_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response_Data); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_tool_grpctool_test_test_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response_Last); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_internal_tool_grpctool_test_test_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Response_First_)(nil),
		(*Response_Data_)(nil),
		(*Response_Last_)(nil),
	}
	file_internal_tool_grpctool_test_test_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*TwoOneofs_M11)(nil),
		(*TwoOneofs_M12)(nil),
		(*TwoOneofs_M21)(nil),
		(*TwoOneofs_M22)(nil),
	}
	file_internal_tool_grpctool_test_test_proto_msgTypes[3].OneofWrappers = []interface{}{
		(*TwoValidOneofs_M11)(nil),
		(*TwoValidOneofs_M12)(nil),
		(*TwoValidOneofs_M21)(nil),
		(*TwoValidOneofs_M22)(nil),
	}
	file_internal_tool_grpctool_test_test_proto_msgTypes[4].OneofWrappers = []interface{}{
		(*OutOfOneof_M1)(nil),
		(*OutOfOneof_M2)(nil),
	}
	file_internal_tool_grpctool_test_test_proto_msgTypes[5].OneofWrappers = []interface{}{
		(*NotAllReachable_M1)(nil),
		(*NotAllReachable_M2)(nil),
		(*NotAllReachable_M3)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_tool_grpctool_test_test_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_tool_grpctool_test_test_proto_goTypes,
		DependencyIndexes: file_internal_tool_grpctool_test_test_proto_depIdxs,
		MessageInfos:      file_internal_tool_grpctool_test_test_proto_msgTypes,
	}.Build()
	File_internal_tool_grpctool_test_test_proto = out.File
	file_internal_tool_grpctool_test_test_proto_rawDesc = nil
	file_internal_tool_grpctool_test_test_proto_goTypes = nil
	file_internal_tool_grpctool_test_test_proto_depIdxs = nil
}
