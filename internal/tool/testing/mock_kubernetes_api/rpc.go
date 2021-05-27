// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/rpc (interfaces: KubernetesApiClient,KubernetesApi_MakeRequestClient)

// Package mock_kubernetes_api is a generated GoMock package.
package mock_kubernetes_api

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	rpc "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/rpc"
	grpctool "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// MockKubernetesApiClient is a mock of KubernetesApiClient interface.
type MockKubernetesApiClient struct {
	ctrl     *gomock.Controller
	recorder *MockKubernetesApiClientMockRecorder
}

// MockKubernetesApiClientMockRecorder is the mock recorder for MockKubernetesApiClient.
type MockKubernetesApiClientMockRecorder struct {
	mock *MockKubernetesApiClient
}

// NewMockKubernetesApiClient creates a new mock instance.
func NewMockKubernetesApiClient(ctrl *gomock.Controller) *MockKubernetesApiClient {
	mock := &MockKubernetesApiClient{ctrl: ctrl}
	mock.recorder = &MockKubernetesApiClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKubernetesApiClient) EXPECT() *MockKubernetesApiClientMockRecorder {
	return m.recorder
}

// MakeRequest mocks base method.
func (m *MockKubernetesApiClient) MakeRequest(arg0 context.Context, arg1 ...grpc.CallOption) (rpc.KubernetesApi_MakeRequestClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "MakeRequest", varargs...)
	ret0, _ := ret[0].(rpc.KubernetesApi_MakeRequestClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MakeRequest indicates an expected call of MakeRequest.
func (mr *MockKubernetesApiClientMockRecorder) MakeRequest(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeRequest", reflect.TypeOf((*MockKubernetesApiClient)(nil).MakeRequest), varargs...)
}

// MockKubernetesApi_MakeRequestClient is a mock of KubernetesApi_MakeRequestClient interface.
type MockKubernetesApi_MakeRequestClient struct {
	ctrl     *gomock.Controller
	recorder *MockKubernetesApi_MakeRequestClientMockRecorder
}

// MockKubernetesApi_MakeRequestClientMockRecorder is the mock recorder for MockKubernetesApi_MakeRequestClient.
type MockKubernetesApi_MakeRequestClientMockRecorder struct {
	mock *MockKubernetesApi_MakeRequestClient
}

// NewMockKubernetesApi_MakeRequestClient creates a new mock instance.
func NewMockKubernetesApi_MakeRequestClient(ctrl *gomock.Controller) *MockKubernetesApi_MakeRequestClient {
	mock := &MockKubernetesApi_MakeRequestClient{ctrl: ctrl}
	mock.recorder = &MockKubernetesApi_MakeRequestClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKubernetesApi_MakeRequestClient) EXPECT() *MockKubernetesApi_MakeRequestClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method.
func (m *MockKubernetesApi_MakeRequestClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockKubernetesApi_MakeRequestClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockKubernetesApi_MakeRequestClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockKubernetesApi_MakeRequestClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockKubernetesApi_MakeRequestClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockKubernetesApi_MakeRequestClient)(nil).Context))
}

// Header mocks base method.
func (m *MockKubernetesApi_MakeRequestClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockKubernetesApi_MakeRequestClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockKubernetesApi_MakeRequestClient)(nil).Header))
}

// Recv mocks base method.
func (m *MockKubernetesApi_MakeRequestClient) Recv() (*grpctool.HttpResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*grpctool.HttpResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockKubernetesApi_MakeRequestClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockKubernetesApi_MakeRequestClient)(nil).Recv))
}

// RecvMsg mocks base method.
func (m *MockKubernetesApi_MakeRequestClient) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockKubernetesApi_MakeRequestClientMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockKubernetesApi_MakeRequestClient)(nil).RecvMsg), arg0)
}

// Send mocks base method.
func (m *MockKubernetesApi_MakeRequestClient) Send(arg0 *grpctool.HttpRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockKubernetesApi_MakeRequestClientMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockKubernetesApi_MakeRequestClient)(nil).Send), arg0)
}

// SendMsg mocks base method.
func (m *MockKubernetesApi_MakeRequestClient) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockKubernetesApi_MakeRequestClientMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockKubernetesApi_MakeRequestClient)(nil).SendMsg), arg0)
}

// Trailer mocks base method.
func (m *MockKubernetesApi_MakeRequestClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockKubernetesApi_MakeRequestClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockKubernetesApi_MakeRequestClient)(nil).Trailer))
}
