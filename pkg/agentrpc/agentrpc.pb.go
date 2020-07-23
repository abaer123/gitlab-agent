// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.24.0
// 	protoc        v3.11.3
// source: pkg/agentrpc/agentrpc.proto

package agentrpc

import (
	context "context"
	reflect "reflect"
	sync "sync"

	proto "github.com/golang/protobuf/proto"
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
}

func (x *ConfigurationRequest) Reset() {
	*x = ConfigurationRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigurationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigurationRequest) ProtoMessage() {}

func (x *ConfigurationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[0]
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
	return file_pkg_agentrpc_agentrpc_proto_rawDescGZIP(), []int{0}
}

type AgentConfiguration struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Deployments *agentcfg.DeploymentsCF `protobuf:"bytes,1,opt,name=deployments,proto3" json:"deployments,omitempty"`
}

func (x *AgentConfiguration) Reset() {
	*x = AgentConfiguration{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AgentConfiguration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AgentConfiguration) ProtoMessage() {}

func (x *AgentConfiguration) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AgentConfiguration.ProtoReflect.Descriptor instead.
func (*AgentConfiguration) Descriptor() ([]byte, []int) {
	return file_pkg_agentrpc_agentrpc_proto_rawDescGZIP(), []int{1}
}

func (x *AgentConfiguration) GetDeployments() *agentcfg.DeploymentsCF {
	if x != nil {
		return x.Deployments
	}
	return nil
}

type ConfigurationResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Configuration *AgentConfiguration `protobuf:"bytes,1,opt,name=configuration,proto3" json:"configuration,omitempty"`
}

func (x *ConfigurationResponse) Reset() {
	*x = ConfigurationResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigurationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigurationResponse) ProtoMessage() {}

func (x *ConfigurationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[2]
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
	return file_pkg_agentrpc_agentrpc_proto_rawDescGZIP(), []int{2}
}

func (x *ConfigurationResponse) GetConfiguration() *AgentConfiguration {
	if x != nil {
		return x.Configuration
	}
	return nil
}

type ObjectsToSynchronizeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ProjectId string `protobuf:"bytes,1,opt,name=project_id,json=projectId,proto3" json:"project_id,omitempty"`
}

func (x *ObjectsToSynchronizeRequest) Reset() {
	*x = ObjectsToSynchronizeRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ObjectsToSynchronizeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ObjectsToSynchronizeRequest) ProtoMessage() {}

func (x *ObjectsToSynchronizeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ObjectsToSynchronizeRequest.ProtoReflect.Descriptor instead.
func (*ObjectsToSynchronizeRequest) Descriptor() ([]byte, []int) {
	return file_pkg_agentrpc_agentrpc_proto_rawDescGZIP(), []int{3}
}

func (x *ObjectsToSynchronizeRequest) GetProjectId() string {
	if x != nil {
		return x.ProjectId
	}
	return ""
}

type ObjectToSynchronize struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Object []byte `protobuf:"bytes,1,opt,name=object,proto3" json:"object,omitempty"`
}

func (x *ObjectToSynchronize) Reset() {
	*x = ObjectToSynchronize{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ObjectToSynchronize) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ObjectToSynchronize) ProtoMessage() {}

func (x *ObjectToSynchronize) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ObjectToSynchronize.ProtoReflect.Descriptor instead.
func (*ObjectToSynchronize) Descriptor() ([]byte, []int) {
	return file_pkg_agentrpc_agentrpc_proto_rawDescGZIP(), []int{4}
}

func (x *ObjectToSynchronize) GetObject() []byte {
	if x != nil {
		return x.Object
	}
	return nil
}

type ObjectsToSynchronizeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Revision string                 `protobuf:"bytes,1,opt,name=revision,proto3" json:"revision,omitempty"`
	Objects  []*ObjectToSynchronize `protobuf:"bytes,2,rep,name=objects,proto3" json:"objects,omitempty"`
}

func (x *ObjectsToSynchronizeResponse) Reset() {
	*x = ObjectsToSynchronizeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ObjectsToSynchronizeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ObjectsToSynchronizeResponse) ProtoMessage() {}

func (x *ObjectsToSynchronizeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_agentrpc_agentrpc_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ObjectsToSynchronizeResponse.ProtoReflect.Descriptor instead.
func (*ObjectsToSynchronizeResponse) Descriptor() ([]byte, []int) {
	return file_pkg_agentrpc_agentrpc_proto_rawDescGZIP(), []int{5}
}

func (x *ObjectsToSynchronizeResponse) GetRevision() string {
	if x != nil {
		return x.Revision
	}
	return ""
}

func (x *ObjectsToSynchronizeResponse) GetObjects() []*ObjectToSynchronize {
	if x != nil {
		return x.Objects
	}
	return nil
}

var File_pkg_agentrpc_agentrpc_proto protoreflect.FileDescriptor

var file_pkg_agentrpc_agentrpc_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x2f, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x1a, 0x1b, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x63, 0x66, 0x67, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x63, 0x66, 0x67, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x16, 0x0a, 0x14, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x4f, 0x0a, 0x12,
	0x41, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x39, 0x0a, 0x0b, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74,
	0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x63,
	0x66, 0x67, 0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x43, 0x46,
	0x52, 0x0b, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x5b, 0x0a,
	0x15, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x42, 0x0a, 0x0d, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x61, 0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0d, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x3c, 0x0a, 0x1b, 0x4f, 0x62,
	0x6a, 0x65, 0x63, 0x74, 0x73, 0x54, 0x6f, 0x53, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69,
	0x7a, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x72, 0x6f,
	0x6a, 0x65, 0x63, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70,
	0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x64, 0x22, 0x2d, 0x0a, 0x13, 0x4f, 0x62, 0x6a, 0x65,
	0x63, 0x74, 0x54, 0x6f, 0x53, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x06, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x22, 0x73, 0x0a, 0x1c, 0x4f, 0x62, 0x6a, 0x65, 0x63,
	0x74, 0x73, 0x54, 0x6f, 0x53, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x65, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73,
	0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73,
	0x69, 0x6f, 0x6e, 0x12, 0x37, 0x0a, 0x07, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x2e,
	0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x54, 0x6f, 0x53, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e,
	0x69, 0x7a, 0x65, 0x52, 0x07, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x32, 0xd6, 0x01, 0x0a,
	0x0d, 0x47, 0x69, 0x74, 0x4c, 0x61, 0x62, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x57,
	0x0a, 0x10, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x1e, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x1f, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x30, 0x01, 0x12, 0x6c, 0x0a, 0x17, 0x47, 0x65, 0x74, 0x4f, 0x62,
	0x6a, 0x65, 0x63, 0x74, 0x73, 0x54, 0x6f, 0x53, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69,
	0x7a, 0x65, 0x12, 0x25, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x4f, 0x62,
	0x6a, 0x65, 0x63, 0x74, 0x73, 0x54, 0x6f, 0x53, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69,
	0x7a, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x26, 0x2e, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x72, 0x70, 0x63, 0x2e, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x54, 0x6f, 0x53, 0x79,
	0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x00, 0x30, 0x01, 0x42, 0x45, 0x5a, 0x43, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2d, 0x6f, 0x72, 0x67, 0x2f, 0x63,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2d, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2f, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2d, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2f,
	0x70, 0x6b, 0x67, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x72, 0x70, 0x63, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_agentrpc_agentrpc_proto_rawDescOnce sync.Once
	file_pkg_agentrpc_agentrpc_proto_rawDescData = file_pkg_agentrpc_agentrpc_proto_rawDesc
)

func file_pkg_agentrpc_agentrpc_proto_rawDescGZIP() []byte {
	file_pkg_agentrpc_agentrpc_proto_rawDescOnce.Do(func() {
		file_pkg_agentrpc_agentrpc_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_agentrpc_agentrpc_proto_rawDescData)
	})
	return file_pkg_agentrpc_agentrpc_proto_rawDescData
}

var file_pkg_agentrpc_agentrpc_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_pkg_agentrpc_agentrpc_proto_goTypes = []interface{}{
	(*ConfigurationRequest)(nil),         // 0: agentrpc.ConfigurationRequest
	(*AgentConfiguration)(nil),           // 1: agentrpc.AgentConfiguration
	(*ConfigurationResponse)(nil),        // 2: agentrpc.ConfigurationResponse
	(*ObjectsToSynchronizeRequest)(nil),  // 3: agentrpc.ObjectsToSynchronizeRequest
	(*ObjectToSynchronize)(nil),          // 4: agentrpc.ObjectToSynchronize
	(*ObjectsToSynchronizeResponse)(nil), // 5: agentrpc.ObjectsToSynchronizeResponse
	(*agentcfg.DeploymentsCF)(nil),       // 6: agentcfg.DeploymentsCF
}
var file_pkg_agentrpc_agentrpc_proto_depIdxs = []int32{
	6, // 0: agentrpc.AgentConfiguration.deployments:type_name -> agentcfg.DeploymentsCF
	1, // 1: agentrpc.ConfigurationResponse.configuration:type_name -> agentrpc.AgentConfiguration
	4, // 2: agentrpc.ObjectsToSynchronizeResponse.objects:type_name -> agentrpc.ObjectToSynchronize
	0, // 3: agentrpc.GitLabService.GetConfiguration:input_type -> agentrpc.ConfigurationRequest
	3, // 4: agentrpc.GitLabService.GetObjectsToSynchronize:input_type -> agentrpc.ObjectsToSynchronizeRequest
	2, // 5: agentrpc.GitLabService.GetConfiguration:output_type -> agentrpc.ConfigurationResponse
	5, // 6: agentrpc.GitLabService.GetObjectsToSynchronize:output_type -> agentrpc.ObjectsToSynchronizeResponse
	5, // [5:7] is the sub-list for method output_type
	3, // [3:5] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_pkg_agentrpc_agentrpc_proto_init() }
func file_pkg_agentrpc_agentrpc_proto_init() {
	if File_pkg_agentrpc_agentrpc_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_agentrpc_agentrpc_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_pkg_agentrpc_agentrpc_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AgentConfiguration); i {
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
		file_pkg_agentrpc_agentrpc_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
		file_pkg_agentrpc_agentrpc_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ObjectsToSynchronizeRequest); i {
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
		file_pkg_agentrpc_agentrpc_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ObjectToSynchronize); i {
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
		file_pkg_agentrpc_agentrpc_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ObjectsToSynchronizeResponse); i {
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
			RawDescriptor: file_pkg_agentrpc_agentrpc_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_pkg_agentrpc_agentrpc_proto_goTypes,
		DependencyIndexes: file_pkg_agentrpc_agentrpc_proto_depIdxs,
		MessageInfos:      file_pkg_agentrpc_agentrpc_proto_msgTypes,
	}.Build()
	File_pkg_agentrpc_agentrpc_proto = out.File
	file_pkg_agentrpc_agentrpc_proto_rawDesc = nil
	file_pkg_agentrpc_agentrpc_proto_goTypes = nil
	file_pkg_agentrpc_agentrpc_proto_depIdxs = nil
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
	GetConfiguration(ctx context.Context, in *ConfigurationRequest, opts ...grpc.CallOption) (GitLabService_GetConfigurationClient, error)
	GetObjectsToSynchronize(ctx context.Context, in *ObjectsToSynchronizeRequest, opts ...grpc.CallOption) (GitLabService_GetObjectsToSynchronizeClient, error)
}

type gitLabServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewGitLabServiceClient(cc grpc.ClientConnInterface) GitLabServiceClient {
	return &gitLabServiceClient{cc}
}

func (c *gitLabServiceClient) GetConfiguration(ctx context.Context, in *ConfigurationRequest, opts ...grpc.CallOption) (GitLabService_GetConfigurationClient, error) {
	stream, err := c.cc.NewStream(ctx, &_GitLabService_serviceDesc.Streams[0], "/agentrpc.GitLabService/GetConfiguration", opts...)
	if err != nil {
		return nil, err
	}
	x := &gitLabServiceGetConfigurationClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GitLabService_GetConfigurationClient interface {
	Recv() (*ConfigurationResponse, error)
	grpc.ClientStream
}

type gitLabServiceGetConfigurationClient struct {
	grpc.ClientStream
}

func (x *gitLabServiceGetConfigurationClient) Recv() (*ConfigurationResponse, error) {
	m := new(ConfigurationResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *gitLabServiceClient) GetObjectsToSynchronize(ctx context.Context, in *ObjectsToSynchronizeRequest, opts ...grpc.CallOption) (GitLabService_GetObjectsToSynchronizeClient, error) {
	stream, err := c.cc.NewStream(ctx, &_GitLabService_serviceDesc.Streams[1], "/agentrpc.GitLabService/GetObjectsToSynchronize", opts...)
	if err != nil {
		return nil, err
	}
	x := &gitLabServiceGetObjectsToSynchronizeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GitLabService_GetObjectsToSynchronizeClient interface {
	Recv() (*ObjectsToSynchronizeResponse, error)
	grpc.ClientStream
}

type gitLabServiceGetObjectsToSynchronizeClient struct {
	grpc.ClientStream
}

func (x *gitLabServiceGetObjectsToSynchronizeClient) Recv() (*ObjectsToSynchronizeResponse, error) {
	m := new(ObjectsToSynchronizeResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// GitLabServiceServer is the server API for GitLabService service.
type GitLabServiceServer interface {
	GetConfiguration(*ConfigurationRequest, GitLabService_GetConfigurationServer) error
	GetObjectsToSynchronize(*ObjectsToSynchronizeRequest, GitLabService_GetObjectsToSynchronizeServer) error
}

// UnimplementedGitLabServiceServer can be embedded to have forward compatible implementations.
type UnimplementedGitLabServiceServer struct {
}

func (*UnimplementedGitLabServiceServer) GetConfiguration(*ConfigurationRequest, GitLabService_GetConfigurationServer) error {
	return status.Errorf(codes.Unimplemented, "method GetConfiguration not implemented")
}
func (*UnimplementedGitLabServiceServer) GetObjectsToSynchronize(*ObjectsToSynchronizeRequest, GitLabService_GetObjectsToSynchronizeServer) error {
	return status.Errorf(codes.Unimplemented, "method GetObjectsToSynchronize not implemented")
}

func RegisterGitLabServiceServer(s *grpc.Server, srv GitLabServiceServer) {
	s.RegisterService(&_GitLabService_serviceDesc, srv)
}

func _GitLabService_GetConfiguration_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ConfigurationRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GitLabServiceServer).GetConfiguration(m, &gitLabServiceGetConfigurationServer{stream})
}

type GitLabService_GetConfigurationServer interface {
	Send(*ConfigurationResponse) error
	grpc.ServerStream
}

type gitLabServiceGetConfigurationServer struct {
	grpc.ServerStream
}

func (x *gitLabServiceGetConfigurationServer) Send(m *ConfigurationResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _GitLabService_GetObjectsToSynchronize_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ObjectsToSynchronizeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GitLabServiceServer).GetObjectsToSynchronize(m, &gitLabServiceGetObjectsToSynchronizeServer{stream})
}

type GitLabService_GetObjectsToSynchronizeServer interface {
	Send(*ObjectsToSynchronizeResponse) error
	grpc.ServerStream
}

type gitLabServiceGetObjectsToSynchronizeServer struct {
	grpc.ServerStream
}

func (x *gitLabServiceGetObjectsToSynchronizeServer) Send(m *ObjectsToSynchronizeResponse) error {
	return x.ServerStream.SendMsg(m)
}

var _GitLabService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "agentrpc.GitLabService",
	HandlerType: (*GitLabServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetConfiguration",
			Handler:       _GitLabService_GetConfiguration_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetObjectsToSynchronize",
			Handler:       _GitLabService_GetObjectsToSynchronize_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "pkg/agentrpc/agentrpc.proto",
}
