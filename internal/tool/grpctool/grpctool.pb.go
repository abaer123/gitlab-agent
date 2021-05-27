// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.13.0
// source: internal/tool/grpctool/grpctool.proto

package grpctool

import (
	reflect "reflect"
	sync "sync"

	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	any "github.com/golang/protobuf/ptypes/any"
	_ "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool/automata"
	prototool "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type HttpRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Message:
	//	*HttpRequest_Header_
	//	*HttpRequest_Data_
	//	*HttpRequest_Trailer_
	Message isHttpRequest_Message `protobuf_oneof:"message"`
}

func (x *HttpRequest) Reset() {
	*x = HttpRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpRequest) ProtoMessage() {}

func (x *HttpRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpRequest.ProtoReflect.Descriptor instead.
func (*HttpRequest) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_grpctool_proto_rawDescGZIP(), []int{0}
}

func (m *HttpRequest) GetMessage() isHttpRequest_Message {
	if m != nil {
		return m.Message
	}
	return nil
}

func (x *HttpRequest) GetHeader() *HttpRequest_Header {
	if x, ok := x.GetMessage().(*HttpRequest_Header_); ok {
		return x.Header
	}
	return nil
}

func (x *HttpRequest) GetData() *HttpRequest_Data {
	if x, ok := x.GetMessage().(*HttpRequest_Data_); ok {
		return x.Data
	}
	return nil
}

func (x *HttpRequest) GetTrailer() *HttpRequest_Trailer {
	if x, ok := x.GetMessage().(*HttpRequest_Trailer_); ok {
		return x.Trailer
	}
	return nil
}

type isHttpRequest_Message interface {
	isHttpRequest_Message()
}

type HttpRequest_Header_ struct {
	Header *HttpRequest_Header `protobuf:"bytes,1,opt,name=header,proto3,oneof"`
}

type HttpRequest_Data_ struct {
	Data *HttpRequest_Data `protobuf:"bytes,2,opt,name=data,proto3,oneof"`
}

type HttpRequest_Trailer_ struct {
	Trailer *HttpRequest_Trailer `protobuf:"bytes,3,opt,name=trailer,proto3,oneof"`
}

func (*HttpRequest_Header_) isHttpRequest_Message() {}

func (*HttpRequest_Data_) isHttpRequest_Message() {}

func (*HttpRequest_Trailer_) isHttpRequest_Message() {}

type HttpResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Message:
	//	*HttpResponse_Header_
	//	*HttpResponse_Data_
	//	*HttpResponse_Trailer_
	Message isHttpResponse_Message `protobuf_oneof:"message"`
}

func (x *HttpResponse) Reset() {
	*x = HttpResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpResponse) ProtoMessage() {}

func (x *HttpResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpResponse.ProtoReflect.Descriptor instead.
func (*HttpResponse) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_grpctool_proto_rawDescGZIP(), []int{1}
}

func (m *HttpResponse) GetMessage() isHttpResponse_Message {
	if m != nil {
		return m.Message
	}
	return nil
}

func (x *HttpResponse) GetHeader() *HttpResponse_Header {
	if x, ok := x.GetMessage().(*HttpResponse_Header_); ok {
		return x.Header
	}
	return nil
}

func (x *HttpResponse) GetData() *HttpResponse_Data {
	if x, ok := x.GetMessage().(*HttpResponse_Data_); ok {
		return x.Data
	}
	return nil
}

func (x *HttpResponse) GetTrailer() *HttpResponse_Trailer {
	if x, ok := x.GetMessage().(*HttpResponse_Trailer_); ok {
		return x.Trailer
	}
	return nil
}

type isHttpResponse_Message interface {
	isHttpResponse_Message()
}

type HttpResponse_Header_ struct {
	Header *HttpResponse_Header `protobuf:"bytes,1,opt,name=header,proto3,oneof"`
}

type HttpResponse_Data_ struct {
	Data *HttpResponse_Data `protobuf:"bytes,2,opt,name=data,proto3,oneof"`
}

type HttpResponse_Trailer_ struct {
	Trailer *HttpResponse_Trailer `protobuf:"bytes,3,opt,name=trailer,proto3,oneof"`
}

func (*HttpResponse_Header_) isHttpResponse_Message() {}

func (*HttpResponse_Data_) isHttpResponse_Message() {}

func (*HttpResponse_Trailer_) isHttpResponse_Message() {}

type HttpRequest_Header struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Request *prototool.HttpRequest `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
	Extra   *any.Any               `protobuf:"bytes,2,opt,name=extra,proto3" json:"extra,omitempty"`
}

func (x *HttpRequest_Header) Reset() {
	*x = HttpRequest_Header{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpRequest_Header) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpRequest_Header) ProtoMessage() {}

func (x *HttpRequest_Header) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpRequest_Header.ProtoReflect.Descriptor instead.
func (*HttpRequest_Header) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_grpctool_proto_rawDescGZIP(), []int{0, 0}
}

func (x *HttpRequest_Header) GetRequest() *prototool.HttpRequest {
	if x != nil {
		return x.Request
	}
	return nil
}

func (x *HttpRequest_Header) GetExtra() *any.Any {
	if x != nil {
		return x.Extra
	}
	return nil
}

type HttpRequest_Data struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *HttpRequest_Data) Reset() {
	*x = HttpRequest_Data{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpRequest_Data) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpRequest_Data) ProtoMessage() {}

func (x *HttpRequest_Data) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpRequest_Data.ProtoReflect.Descriptor instead.
func (*HttpRequest_Data) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_grpctool_proto_rawDescGZIP(), []int{0, 1}
}

func (x *HttpRequest_Data) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type HttpRequest_Trailer struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *HttpRequest_Trailer) Reset() {
	*x = HttpRequest_Trailer{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpRequest_Trailer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpRequest_Trailer) ProtoMessage() {}

func (x *HttpRequest_Trailer) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpRequest_Trailer.ProtoReflect.Descriptor instead.
func (*HttpRequest_Trailer) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_grpctool_proto_rawDescGZIP(), []int{0, 2}
}

type HttpResponse_Header struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Response *prototool.HttpResponse `protobuf:"bytes,1,opt,name=response,proto3" json:"response,omitempty"`
}

func (x *HttpResponse_Header) Reset() {
	*x = HttpResponse_Header{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpResponse_Header) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpResponse_Header) ProtoMessage() {}

func (x *HttpResponse_Header) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpResponse_Header.ProtoReflect.Descriptor instead.
func (*HttpResponse_Header) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_grpctool_proto_rawDescGZIP(), []int{1, 0}
}

func (x *HttpResponse_Header) GetResponse() *prototool.HttpResponse {
	if x != nil {
		return x.Response
	}
	return nil
}

type HttpResponse_Data struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *HttpResponse_Data) Reset() {
	*x = HttpResponse_Data{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpResponse_Data) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpResponse_Data) ProtoMessage() {}

func (x *HttpResponse_Data) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpResponse_Data.ProtoReflect.Descriptor instead.
func (*HttpResponse_Data) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_grpctool_proto_rawDescGZIP(), []int{1, 1}
}

func (x *HttpResponse_Data) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type HttpResponse_Trailer struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *HttpResponse_Trailer) Reset() {
	*x = HttpResponse_Trailer{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpResponse_Trailer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpResponse_Trailer) ProtoMessage() {}

func (x *HttpResponse_Trailer) ProtoReflect() protoreflect.Message {
	mi := &file_internal_tool_grpctool_grpctool_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpResponse_Trailer.ProtoReflect.Descriptor instead.
func (*HttpResponse_Trailer) Descriptor() ([]byte, []int) {
	return file_internal_tool_grpctool_grpctool_proto_rawDescGZIP(), []int{1, 2}
}

var File_internal_tool_grpctool_grpctool_proto protoreflect.FileDescriptor

var file_internal_tool_grpctool_grpctool_proto_rawDesc = []byte{
	0x0a, 0x25, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x6f, 0x6f, 0x6c, 0x2f,
	0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f,
	0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e,
	0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x1a, 0x2e,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x67, 0x72,
	0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x61, 0x75, 0x74, 0x6f, 0x6d, 0x61, 0x74, 0x61, 0x2f,
	0x61, 0x75, 0x74, 0x6f, 0x6d, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x27,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x74, 0x6f, 0x6f,
	0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x19, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xb6, 0x03, 0x0a, 0x0b,
	0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x4d, 0x0a, 0x06, 0x68,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x67, 0x69,
	0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74,
	0x6f, 0x6f, 0x6c, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e,
	0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x42, 0x08, 0x80, 0xf6, 0x2c, 0x02, 0x80, 0xf6, 0x2c, 0x03,
	0x48, 0x00, 0x52, 0x06, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x47, 0x0a, 0x04, 0x64, 0x61,
	0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61,
	0x62, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c,
	0x2e, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x44, 0x61, 0x74,
	0x61, 0x42, 0x08, 0x80, 0xf6, 0x2c, 0x02, 0x80, 0xf6, 0x2c, 0x03, 0x48, 0x00, 0x52, 0x04, 0x64,
	0x61, 0x74, 0x61, 0x12, 0x55, 0x0a, 0x07, 0x74, 0x72, 0x61, 0x69, 0x6c, 0x65, 0x72, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67,
	0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x2e, 0x48, 0x74, 0x74,
	0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x54, 0x72, 0x61, 0x69, 0x6c, 0x65, 0x72,
	0x42, 0x0d, 0x80, 0xf6, 0x2c, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x48,
	0x00, 0x52, 0x07, 0x74, 0x72, 0x61, 0x69, 0x6c, 0x65, 0x72, 0x1a, 0x7d, 0x0a, 0x06, 0x48, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x12, 0x47, 0x0a, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x74, 0x6f, 0x6f, 0x6c, 0x2e, 0x48,
	0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a,
	0x01, 0x02, 0x10, 0x01, 0x52, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2a, 0x0a,
	0x05, 0x65, 0x78, 0x74, 0x72, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41,
	0x6e, 0x79, 0x52, 0x05, 0x65, 0x78, 0x74, 0x72, 0x61, 0x1a, 0x1a, 0x0a, 0x04, 0x44, 0x61, 0x74,
	0x61, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x1a, 0x09, 0x0a, 0x07, 0x54, 0x72, 0x61, 0x69, 0x6c, 0x65, 0x72,
	0x42, 0x12, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x07, 0x88, 0xf6, 0x2c,
	0x01, 0xf8, 0x42, 0x01, 0x22, 0x91, 0x03, 0x0a, 0x0c, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4e, 0x0a, 0x06, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x2e, 0x48, 0x74,
	0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x42, 0x08, 0x80, 0xf6, 0x2c, 0x02, 0x80, 0xf6, 0x2c, 0x03, 0x48, 0x00, 0x52, 0x06, 0x68,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x48, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x2e, 0x48, 0x74, 0x74, 0x70,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x42, 0x08, 0x80,
	0xf6, 0x2c, 0x02, 0x80, 0xf6, 0x2c, 0x03, 0x48, 0x00, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12,
	0x56, 0x0a, 0x07, 0x74, 0x72, 0x61, 0x69, 0x6c, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x2b, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e,
	0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x54, 0x72, 0x61, 0x69, 0x6c, 0x65, 0x72, 0x42, 0x0d, 0x80,
	0xf6, 0x2c, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x48, 0x00, 0x52, 0x07,
	0x74, 0x72, 0x61, 0x69, 0x6c, 0x65, 0x72, 0x1a, 0x54, 0x0a, 0x06, 0x48, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x12, 0x4a, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x74, 0x6f, 0x6f, 0x6c, 0x2e, 0x48, 0x74, 0x74,
	0x70, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01,
	0x02, 0x10, 0x01, 0x52, 0x08, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x1a, 0x1a, 0x0a,
	0x04, 0x44, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x1a, 0x09, 0x0a, 0x07, 0x54, 0x72, 0x61,
	0x69, 0x6c, 0x65, 0x72, 0x42, 0x12, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12,
	0x07, 0x88, 0xf6, 0x2c, 0x01, 0xf8, 0x42, 0x01, 0x42, 0x53, 0x5a, 0x51, 0x67, 0x69, 0x74, 0x6c,
	0x61, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2d, 0x6f, 0x72,
	0x67, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2d, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2d, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x2f, 0x76, 0x31, 0x34, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x74, 0x6f, 0x6f, 0x6c, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x74, 0x6f, 0x6f, 0x6c, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_tool_grpctool_grpctool_proto_rawDescOnce sync.Once
	file_internal_tool_grpctool_grpctool_proto_rawDescData = file_internal_tool_grpctool_grpctool_proto_rawDesc
)

func file_internal_tool_grpctool_grpctool_proto_rawDescGZIP() []byte {
	file_internal_tool_grpctool_grpctool_proto_rawDescOnce.Do(func() {
		file_internal_tool_grpctool_grpctool_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_tool_grpctool_grpctool_proto_rawDescData)
	})
	return file_internal_tool_grpctool_grpctool_proto_rawDescData
}

var file_internal_tool_grpctool_grpctool_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_internal_tool_grpctool_grpctool_proto_goTypes = []interface{}{
	(*HttpRequest)(nil),            // 0: gitlab.agent.grpctool.HttpRequest
	(*HttpResponse)(nil),           // 1: gitlab.agent.grpctool.HttpResponse
	(*HttpRequest_Header)(nil),     // 2: gitlab.agent.grpctool.HttpRequest.Header
	(*HttpRequest_Data)(nil),       // 3: gitlab.agent.grpctool.HttpRequest.Data
	(*HttpRequest_Trailer)(nil),    // 4: gitlab.agent.grpctool.HttpRequest.Trailer
	(*HttpResponse_Header)(nil),    // 5: gitlab.agent.grpctool.HttpResponse.Header
	(*HttpResponse_Data)(nil),      // 6: gitlab.agent.grpctool.HttpResponse.Data
	(*HttpResponse_Trailer)(nil),   // 7: gitlab.agent.grpctool.HttpResponse.Trailer
	(*prototool.HttpRequest)(nil),  // 8: gitlab.agent.prototool.HttpRequest
	(*any.Any)(nil),                // 9: google.protobuf.Any
	(*prototool.HttpResponse)(nil), // 10: gitlab.agent.prototool.HttpResponse
}
var file_internal_tool_grpctool_grpctool_proto_depIdxs = []int32{
	2,  // 0: gitlab.agent.grpctool.HttpRequest.header:type_name -> gitlab.agent.grpctool.HttpRequest.Header
	3,  // 1: gitlab.agent.grpctool.HttpRequest.data:type_name -> gitlab.agent.grpctool.HttpRequest.Data
	4,  // 2: gitlab.agent.grpctool.HttpRequest.trailer:type_name -> gitlab.agent.grpctool.HttpRequest.Trailer
	5,  // 3: gitlab.agent.grpctool.HttpResponse.header:type_name -> gitlab.agent.grpctool.HttpResponse.Header
	6,  // 4: gitlab.agent.grpctool.HttpResponse.data:type_name -> gitlab.agent.grpctool.HttpResponse.Data
	7,  // 5: gitlab.agent.grpctool.HttpResponse.trailer:type_name -> gitlab.agent.grpctool.HttpResponse.Trailer
	8,  // 6: gitlab.agent.grpctool.HttpRequest.Header.request:type_name -> gitlab.agent.prototool.HttpRequest
	9,  // 7: gitlab.agent.grpctool.HttpRequest.Header.extra:type_name -> google.protobuf.Any
	10, // 8: gitlab.agent.grpctool.HttpResponse.Header.response:type_name -> gitlab.agent.prototool.HttpResponse
	9,  // [9:9] is the sub-list for method output_type
	9,  // [9:9] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_internal_tool_grpctool_grpctool_proto_init() }
func file_internal_tool_grpctool_grpctool_proto_init() {
	if File_internal_tool_grpctool_grpctool_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_tool_grpctool_grpctool_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpRequest); i {
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
		file_internal_tool_grpctool_grpctool_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpResponse); i {
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
		file_internal_tool_grpctool_grpctool_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpRequest_Header); i {
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
		file_internal_tool_grpctool_grpctool_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpRequest_Data); i {
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
		file_internal_tool_grpctool_grpctool_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpRequest_Trailer); i {
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
		file_internal_tool_grpctool_grpctool_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpResponse_Header); i {
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
		file_internal_tool_grpctool_grpctool_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpResponse_Data); i {
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
		file_internal_tool_grpctool_grpctool_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpResponse_Trailer); i {
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
	file_internal_tool_grpctool_grpctool_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*HttpRequest_Header_)(nil),
		(*HttpRequest_Data_)(nil),
		(*HttpRequest_Trailer_)(nil),
	}
	file_internal_tool_grpctool_grpctool_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*HttpResponse_Header_)(nil),
		(*HttpResponse_Data_)(nil),
		(*HttpResponse_Trailer_)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_tool_grpctool_grpctool_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_tool_grpctool_grpctool_proto_goTypes,
		DependencyIndexes: file_internal_tool_grpctool_grpctool_proto_depIdxs,
		MessageInfos:      file_internal_tool_grpctool_grpctool_proto_msgTypes,
	}.Build()
	File_internal_tool_grpctool_grpctool_proto = out.File
	file_internal_tool_grpctool_grpctool_proto_rawDesc = nil
	file_internal_tool_grpctool_grpctool_proto_goTypes = nil
	file_internal_tool_grpctool_grpctool_proto_depIdxs = nil
}
