// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool (interfaces: InboundGrpcToOutboundHttpStream)

// Package mock_rpc is a generated GoMock package.
package mock_rpc

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	grpctool "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	metadata "google.golang.org/grpc/metadata"
)

// MockInboundGrpcToOutboundHttpStream is a mock of InboundGrpcToOutboundHttpStream interface.
type MockInboundGrpcToOutboundHttpStream struct {
	ctrl     *gomock.Controller
	recorder *MockInboundGrpcToOutboundHttpStreamMockRecorder
}

// MockInboundGrpcToOutboundHttpStreamMockRecorder is the mock recorder for MockInboundGrpcToOutboundHttpStream.
type MockInboundGrpcToOutboundHttpStreamMockRecorder struct {
	mock *MockInboundGrpcToOutboundHttpStream
}

// NewMockInboundGrpcToOutboundHttpStream creates a new mock instance.
func NewMockInboundGrpcToOutboundHttpStream(ctrl *gomock.Controller) *MockInboundGrpcToOutboundHttpStream {
	mock := &MockInboundGrpcToOutboundHttpStream{ctrl: ctrl}
	mock.recorder = &MockInboundGrpcToOutboundHttpStreamMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInboundGrpcToOutboundHttpStream) EXPECT() *MockInboundGrpcToOutboundHttpStreamMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockInboundGrpcToOutboundHttpStream) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockInboundGrpcToOutboundHttpStreamMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockInboundGrpcToOutboundHttpStream)(nil).Context))
}

// RecvMsg mocks base method.
func (m *MockInboundGrpcToOutboundHttpStream) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockInboundGrpcToOutboundHttpStreamMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockInboundGrpcToOutboundHttpStream)(nil).RecvMsg), arg0)
}

// Send mocks base method.
func (m *MockInboundGrpcToOutboundHttpStream) Send(arg0 *grpctool.HttpResponse) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockInboundGrpcToOutboundHttpStreamMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockInboundGrpcToOutboundHttpStream)(nil).Send), arg0)
}

// SendHeader mocks base method.
func (m *MockInboundGrpcToOutboundHttpStream) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader.
func (mr *MockInboundGrpcToOutboundHttpStreamMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockInboundGrpcToOutboundHttpStream)(nil).SendHeader), arg0)
}

// SendMsg mocks base method.
func (m *MockInboundGrpcToOutboundHttpStream) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockInboundGrpcToOutboundHttpStreamMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockInboundGrpcToOutboundHttpStream)(nil).SendMsg), arg0)
}

// SetHeader mocks base method.
func (m *MockInboundGrpcToOutboundHttpStream) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader.
func (mr *MockInboundGrpcToOutboundHttpStreamMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockInboundGrpcToOutboundHttpStream)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method.
func (m *MockInboundGrpcToOutboundHttpStream) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer.
func (mr *MockInboundGrpcToOutboundHttpStreamMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockInboundGrpcToOutboundHttpStream)(nil).SetTrailer), arg0)
}
