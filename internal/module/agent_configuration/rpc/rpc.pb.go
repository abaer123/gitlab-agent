// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.13.0
// source: internal/module/agent_configuration/rpc/rpc.proto

package rpc

import (
	context "context"
	reflect "reflect"
	sync "sync"

	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/golang/protobuf/proto"
	modshared "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	agentcfg "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type ConfigurationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommitId  string               `protobuf:"bytes,1,opt,name=commit_id,json=commitId,proto3" json:"commit_id,omitempty"`
	AgentMeta *modshared.AgentMeta `protobuf:"bytes,2,opt,name=agent_meta,json=agentMeta,proto3" json:"agent_meta,omitempty"`
}

func (x *ConfigurationRequest) Reset() {
	*x = ConfigurationRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_module_agent_configuration_rpc_rpc_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigurationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigurationRequest) ProtoMessage() {}

func (x *ConfigurationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_module_agent_configuration_rpc_rpc_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigurationRequest.ProtoReflect.Descriptor instead.
func (*ConfigurationRequest) Descriptor() ([]byte, []int) {
	return file_internal_module_agent_configuration_rpc_rpc_proto_rawDescGZIP(), []int{0}
}

func (x *ConfigurationRequest) GetCommitId() string {
	if x != nil {
		return x.CommitId
	}
	return ""
}

func (x *ConfigurationRequest) GetAgentMeta() *modshared.AgentMeta {
	if x != nil {
		return x.AgentMeta
	}
	return nil
}

type ConfigurationResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Configuration *agentcfg.AgentConfiguration `protobuf:"bytes,1,opt,name=configuration,proto3" json:"configuration,omitempty"`
	CommitId      string                       `protobuf:"bytes,2,opt,name=commit_id,json=commitId,proto3" json:"commit_id,omitempty"`
}

func (x *ConfigurationResponse) Reset() {
	*x = ConfigurationResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_module_agent_configuration_rpc_rpc_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigurationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigurationResponse) ProtoMessage() {}

func (x *ConfigurationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_module_agent_configuration_rpc_rpc_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigurationResponse.ProtoReflect.Descriptor instead.
func (*ConfigurationResponse) Descriptor() ([]byte, []int) {
	return file_internal_module_agent_configuration_rpc_rpc_proto_rawDescGZIP(), []int{1}
}

func (x *ConfigurationResponse) GetConfiguration() *agentcfg.AgentConfiguration {
	if x != nil {
		return x.Configuration
	}
	return nil
}

func (x *ConfigurationResponse) GetCommitId() string {
	if x != nil {
		return x.CommitId
	}
	return ""
}

var File_internal_module_agent_configuration_rpc_rpc_proto protoreflect.FileDescriptor

var file_internal_module_agent_configuration_rpc_rpc_proto_rawDesc = []byte{
	0x0a, 0x31, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c,
	0x65, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x24, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x72, 0x70, 0x63, 0x1a, 0x1b, 0x70, 0x6b, 0x67, 0x2f, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x63, 0x66, 0x67, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x63, 0x66, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x29, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c,
	0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2f, 0x6d, 0x6f, 0x64, 0x73, 0x68, 0x61, 0x72, 0x65,
	0x64, 0x2f, 0x6d, 0x6f, 0x64, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69,
	0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x75, 0x0a, 0x14, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x64, 0x12,
	0x40, 0x0a, 0x0a, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x2e, 0x6d, 0x6f, 0x64, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x2e, 0x41, 0x67, 0x65,
	0x6e, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x52, 0x09, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x4d, 0x65, 0x74,
	0x61, 0x22, 0x8e, 0x01, 0x0a, 0x15, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4f, 0x0a, 0x0d, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x29, 0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x63, 0x66, 0x67, 0x2e, 0x41, 0x67, 0x65, 0x6e, 0x74,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0d, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x24, 0x0a, 0x09,
	0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42,
	0x07, 0xfa, 0x42, 0x04, 0x72, 0x02, 0x10, 0x01, 0x52, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x49, 0x64, 0x32, 0xa6, 0x01, 0x0a, 0x12, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x8f, 0x01, 0x0a, 0x10, 0x47, 0x65,
	0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x3a,
	0x2e, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x67,
	0x65, 0x6e, 0x74, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x3b, 0x2e, 0x67, 0x69, 0x74,
	0x6c, 0x61, 0x62, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x72, 0x70,
	0x63, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x30, 0x01, 0x42, 0x60, 0x5a, 0x5e, 0x67,
	0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62,
	0x2d, 0x6f, 0x72, 0x67, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2d, 0x69, 0x6e, 0x74,
	0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2d,
	0x61, 0x67, 0x65, 0x6e, 0x74, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6d,
	0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x72, 0x70, 0x63, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_module_agent_configuration_rpc_rpc_proto_rawDescOnce sync.Once
	file_internal_module_agent_configuration_rpc_rpc_proto_rawDescData = file_internal_module_agent_configuration_rpc_rpc_proto_rawDesc
)

func file_internal_module_agent_configuration_rpc_rpc_proto_rawDescGZIP() []byte {
	file_internal_module_agent_configuration_rpc_rpc_proto_rawDescOnce.Do(func() {
		file_internal_module_agent_configuration_rpc_rpc_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_module_agent_configuration_rpc_rpc_proto_rawDescData)
	})
	return file_internal_module_agent_configuration_rpc_rpc_proto_rawDescData
}

var file_internal_module_agent_configuration_rpc_rpc_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_internal_module_agent_configuration_rpc_rpc_proto_goTypes = []interface{}{
	(*ConfigurationRequest)(nil),        // 0: gitlab.agent.agent_configuration.rpc.ConfigurationRequest
	(*ConfigurationResponse)(nil),       // 1: gitlab.agent.agent_configuration.rpc.ConfigurationResponse
	(*modshared.AgentMeta)(nil),         // 2: gitlab.agent.modshared.AgentMeta
	(*agentcfg.AgentConfiguration)(nil), // 3: gitlab.agent.agentcfg.AgentConfiguration
}
var file_internal_module_agent_configuration_rpc_rpc_proto_depIdxs = []int32{
	2, // 0: gitlab.agent.agent_configuration.rpc.ConfigurationRequest.agent_meta:type_name -> gitlab.agent.modshared.AgentMeta
	3, // 1: gitlab.agent.agent_configuration.rpc.ConfigurationResponse.configuration:type_name -> gitlab.agent.agentcfg.AgentConfiguration
	0, // 2: gitlab.agent.agent_configuration.rpc.AgentConfiguration.GetConfiguration:input_type -> gitlab.agent.agent_configuration.rpc.ConfigurationRequest
	1, // 3: gitlab.agent.agent_configuration.rpc.AgentConfiguration.GetConfiguration:output_type -> gitlab.agent.agent_configuration.rpc.ConfigurationResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_internal_module_agent_configuration_rpc_rpc_proto_init() }
func file_internal_module_agent_configuration_rpc_rpc_proto_init() {
	if File_internal_module_agent_configuration_rpc_rpc_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_module_agent_configuration_rpc_rpc_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConfigurationRequest); i {
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
		file_internal_module_agent_configuration_rpc_rpc_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConfigurationResponse); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_module_agent_configuration_rpc_rpc_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_internal_module_agent_configuration_rpc_rpc_proto_goTypes,
		DependencyIndexes: file_internal_module_agent_configuration_rpc_rpc_proto_depIdxs,
		MessageInfos:      file_internal_module_agent_configuration_rpc_rpc_proto_msgTypes,
	}.Build()
	File_internal_module_agent_configuration_rpc_rpc_proto = out.File
	file_internal_module_agent_configuration_rpc_rpc_proto_rawDesc = nil
	file_internal_module_agent_configuration_rpc_rpc_proto_goTypes = nil
	file_internal_module_agent_configuration_rpc_rpc_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// AgentConfigurationClient is the client API for AgentConfiguration service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AgentConfigurationClient interface {
	GetConfiguration(ctx context.Context, in *ConfigurationRequest, opts ...grpc.CallOption) (AgentConfiguration_GetConfigurationClient, error)
}

type agentConfigurationClient struct {
	cc grpc.ClientConnInterface
}

func NewAgentConfigurationClient(cc grpc.ClientConnInterface) AgentConfigurationClient {
	return &agentConfigurationClient{cc}
}

func (c *agentConfigurationClient) GetConfiguration(ctx context.Context, in *ConfigurationRequest, opts ...grpc.CallOption) (AgentConfiguration_GetConfigurationClient, error) {
	stream, err := c.cc.NewStream(ctx, &_AgentConfiguration_serviceDesc.Streams[0], "/gitlab.agent.agent_configuration.rpc.AgentConfiguration/GetConfiguration", opts...)
	if err != nil {
		return nil, err
	}
	x := &agentConfigurationGetConfigurationClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AgentConfiguration_GetConfigurationClient interface {
	Recv() (*ConfigurationResponse, error)
	grpc.ClientStream
}

type agentConfigurationGetConfigurationClient struct {
	grpc.ClientStream
}

func (x *agentConfigurationGetConfigurationClient) Recv() (*ConfigurationResponse, error) {
	m := new(ConfigurationResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// AgentConfigurationServer is the server API for AgentConfiguration service.
type AgentConfigurationServer interface {
	GetConfiguration(*ConfigurationRequest, AgentConfiguration_GetConfigurationServer) error
}

// UnimplementedAgentConfigurationServer can be embedded to have forward compatible implementations.
type UnimplementedAgentConfigurationServer struct {
}

func (*UnimplementedAgentConfigurationServer) GetConfiguration(*ConfigurationRequest, AgentConfiguration_GetConfigurationServer) error {
	return status.Errorf(codes.Unimplemented, "method GetConfiguration not implemented")
}

func RegisterAgentConfigurationServer(s *grpc.Server, srv AgentConfigurationServer) {
	s.RegisterService(&_AgentConfiguration_serviceDesc, srv)
}

func _AgentConfiguration_GetConfiguration_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ConfigurationRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AgentConfigurationServer).GetConfiguration(m, &agentConfigurationGetConfigurationServer{stream})
}

type AgentConfiguration_GetConfigurationServer interface {
	Send(*ConfigurationResponse) error
	grpc.ServerStream
}

type agentConfigurationGetConfigurationServer struct {
	grpc.ServerStream
}

func (x *agentConfigurationGetConfigurationServer) Send(m *ConfigurationResponse) error {
	return x.ServerStream.SendMsg(m)
}

var _AgentConfiguration_serviceDesc = grpc.ServiceDesc{
	ServiceName: "gitlab.agent.agent_configuration.rpc.AgentConfiguration",
	HandlerType: (*AgentConfigurationServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetConfiguration",
			Handler:       _AgentConfiguration_GetConfiguration_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "internal/module/agent_configuration/rpc/rpc.proto",
}
