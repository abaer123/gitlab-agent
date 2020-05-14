// Code generated by protoc-gen-go. DO NOT EDIT.
// source: pkg/agentrpc/agentrpc.proto

package agentrpc

import (
	context "context"
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type ConfigurationRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ConfigurationRequest) Reset()         { *m = ConfigurationRequest{} }
func (m *ConfigurationRequest) String() string { return proto.CompactTextString(m) }
func (*ConfigurationRequest) ProtoMessage()    {}
func (*ConfigurationRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_213b842ecf18ef32, []int{0}
}

func (m *ConfigurationRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ConfigurationRequest.Unmarshal(m, b)
}
func (m *ConfigurationRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ConfigurationRequest.Marshal(b, m, deterministic)
}
func (m *ConfigurationRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ConfigurationRequest.Merge(m, src)
}
func (m *ConfigurationRequest) XXX_Size() int {
	return xxx_messageInfo_ConfigurationRequest.Size(m)
}
func (m *ConfigurationRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ConfigurationRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ConfigurationRequest proto.InternalMessageInfo

type ConfigurationResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ConfigurationResponse) Reset()         { *m = ConfigurationResponse{} }
func (m *ConfigurationResponse) String() string { return proto.CompactTextString(m) }
func (*ConfigurationResponse) ProtoMessage()    {}
func (*ConfigurationResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_213b842ecf18ef32, []int{1}
}

func (m *ConfigurationResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ConfigurationResponse.Unmarshal(m, b)
}
func (m *ConfigurationResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ConfigurationResponse.Marshal(b, m, deterministic)
}
func (m *ConfigurationResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ConfigurationResponse.Merge(m, src)
}
func (m *ConfigurationResponse) XXX_Size() int {
	return xxx_messageInfo_ConfigurationResponse.Size(m)
}
func (m *ConfigurationResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ConfigurationResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ConfigurationResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*ConfigurationRequest)(nil), "agentrpc.ConfigurationRequest")
	proto.RegisterType((*ConfigurationResponse)(nil), "agentrpc.ConfigurationResponse")
}

func init() { proto.RegisterFile("pkg/agentrpc/agentrpc.proto", fileDescriptor_213b842ecf18ef32) }

var fileDescriptor_213b842ecf18ef32 = []byte{
	// 137 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x2e, 0xc8, 0x4e, 0xd7,
	0x4f, 0x4c, 0x4f, 0xcd, 0x2b, 0x29, 0x2a, 0x48, 0x86, 0x33, 0xf4, 0x0a, 0x8a, 0xf2, 0x4b, 0xf2,
	0x85, 0x38, 0x60, 0x7c, 0x25, 0x31, 0x2e, 0x11, 0xe7, 0xfc, 0xbc, 0xb4, 0xcc, 0xf4, 0xd2, 0xa2,
	0xc4, 0x92, 0xcc, 0xfc, 0xbc, 0xa0, 0xd4, 0xc2, 0xd2, 0xd4, 0xe2, 0x12, 0x25, 0x71, 0x2e, 0x51,
	0x34, 0xf1, 0xe2, 0x82, 0xfc, 0xbc, 0xe2, 0x54, 0xa3, 0x34, 0x2e, 0x5e, 0xf7, 0xcc, 0x12, 0x9f,
	0xc4, 0xa4, 0xe0, 0xd4, 0xa2, 0xb2, 0xcc, 0xe4, 0x54, 0xa1, 0x50, 0x2e, 0x01, 0xf7, 0xd4, 0x12,
	0xb8, 0xe2, 0xcc, 0x92, 0xfc, 0x3c, 0x21, 0x39, 0x3d, 0xb8, 0x85, 0xd8, 0x4c, 0x97, 0x92, 0xc7,
	0x29, 0x0f, 0xb1, 0x45, 0x89, 0x21, 0x89, 0x0d, 0xec, 0x52, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff,
	0xff, 0xf6, 0x83, 0xa3, 0x9a, 0xc8, 0x00, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// GitLabServiceClient is the client API for GitLabService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type GitLabServiceClient interface {
	GetConfiguraiton(ctx context.Context, in *ConfigurationRequest, opts ...grpc.CallOption) (*ConfigurationResponse, error)
}

type gitLabServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewGitLabServiceClient(cc grpc.ClientConnInterface) GitLabServiceClient {
	return &gitLabServiceClient{cc}
}

func (c *gitLabServiceClient) GetConfiguraiton(ctx context.Context, in *ConfigurationRequest, opts ...grpc.CallOption) (*ConfigurationResponse, error) {
	out := new(ConfigurationResponse)
	err := c.cc.Invoke(ctx, "/agentrpc.GitLabService/GetConfiguraiton", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GitLabServiceServer is the server API for GitLabService service.
type GitLabServiceServer interface {
	GetConfiguraiton(context.Context, *ConfigurationRequest) (*ConfigurationResponse, error)
}

// UnimplementedGitLabServiceServer can be embedded to have forward compatible implementations.
type UnimplementedGitLabServiceServer struct {
}

func (*UnimplementedGitLabServiceServer) GetConfiguraiton(ctx context.Context, req *ConfigurationRequest) (*ConfigurationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfiguraiton not implemented")
}

func RegisterGitLabServiceServer(s *grpc.Server, srv GitLabServiceServer) {
	s.RegisterService(&_GitLabService_serviceDesc, srv)
}

func _GitLabService_GetConfiguraiton_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConfigurationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GitLabServiceServer).GetConfiguraiton(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agentrpc.GitLabService/GetConfiguraiton",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GitLabServiceServer).GetConfiguraiton(ctx, req.(*ConfigurationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _GitLabService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "agentrpc.GitLabService",
	HandlerType: (*GitLabServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetConfiguraiton",
			Handler:    _GitLabService_GetConfiguraiton_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/agentrpc/agentrpc.proto",
}
