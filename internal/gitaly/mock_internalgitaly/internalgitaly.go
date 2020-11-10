// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly (interfaces: PoolInterface,FetchVisitor,PathEntryVisitor)

// Package mock_internalgitaly is a generated GoMock package.
package mock_internalgitaly

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	api "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	gitaly "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	gitalypb "gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// MockPoolInterface is a mock of PoolInterface interface
type MockPoolInterface struct {
	ctrl     *gomock.Controller
	recorder *MockPoolInterfaceMockRecorder
}

// MockPoolInterfaceMockRecorder is the mock recorder for MockPoolInterface
type MockPoolInterfaceMockRecorder struct {
	mock *MockPoolInterface
}

// NewMockPoolInterface creates a new mock instance
func NewMockPoolInterface(ctrl *gomock.Controller) *MockPoolInterface {
	mock := &MockPoolInterface{ctrl: ctrl}
	mock.recorder = &MockPoolInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPoolInterface) EXPECT() *MockPoolInterfaceMockRecorder {
	return m.recorder
}

// CommitServiceClient mocks base method
func (m *MockPoolInterface) CommitServiceClient(arg0 context.Context, arg1 *api.GitalyInfo) (gitalypb.CommitServiceClient, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CommitServiceClient", arg0, arg1)
	ret0, _ := ret[0].(gitalypb.CommitServiceClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CommitServiceClient indicates an expected call of CommitServiceClient
func (mr *MockPoolInterfaceMockRecorder) CommitServiceClient(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitServiceClient", reflect.TypeOf((*MockPoolInterface)(nil).CommitServiceClient), arg0, arg1)
}

// SmartHTTPServiceClient mocks base method
func (m *MockPoolInterface) SmartHTTPServiceClient(arg0 context.Context, arg1 *api.GitalyInfo) (gitalypb.SmartHTTPServiceClient, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SmartHTTPServiceClient", arg0, arg1)
	ret0, _ := ret[0].(gitalypb.SmartHTTPServiceClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SmartHTTPServiceClient indicates an expected call of SmartHTTPServiceClient
func (mr *MockPoolInterfaceMockRecorder) SmartHTTPServiceClient(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SmartHTTPServiceClient", reflect.TypeOf((*MockPoolInterface)(nil).SmartHTTPServiceClient), arg0, arg1)
}

// MockFetchVisitor is a mock of FetchVisitor interface
type MockFetchVisitor struct {
	ctrl     *gomock.Controller
	recorder *MockFetchVisitorMockRecorder
}

// MockFetchVisitorMockRecorder is the mock recorder for MockFetchVisitor
type MockFetchVisitorMockRecorder struct {
	mock *MockFetchVisitor
}

// NewMockFetchVisitor creates a new mock instance
func NewMockFetchVisitor(ctrl *gomock.Controller) *MockFetchVisitor {
	mock := &MockFetchVisitor{ctrl: ctrl}
	mock.recorder = &MockFetchVisitorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFetchVisitor) EXPECT() *MockFetchVisitorMockRecorder {
	return m.recorder
}

// VisitBlob mocks base method
func (m *MockFetchVisitor) VisitBlob(arg0 gitaly.Blob) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VisitBlob", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VisitBlob indicates an expected call of VisitBlob
func (mr *MockFetchVisitorMockRecorder) VisitBlob(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VisitBlob", reflect.TypeOf((*MockFetchVisitor)(nil).VisitBlob), arg0)
}

// VisitEntry mocks base method
func (m *MockFetchVisitor) VisitEntry(arg0 *gitalypb.TreeEntry) (bool, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VisitEntry", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// VisitEntry indicates an expected call of VisitEntry
func (mr *MockFetchVisitorMockRecorder) VisitEntry(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VisitEntry", reflect.TypeOf((*MockFetchVisitor)(nil).VisitEntry), arg0)
}

// MockPathEntryVisitor is a mock of PathEntryVisitor interface
type MockPathEntryVisitor struct {
	ctrl     *gomock.Controller
	recorder *MockPathEntryVisitorMockRecorder
}

// MockPathEntryVisitorMockRecorder is the mock recorder for MockPathEntryVisitor
type MockPathEntryVisitorMockRecorder struct {
	mock *MockPathEntryVisitor
}

// NewMockPathEntryVisitor creates a new mock instance
func NewMockPathEntryVisitor(ctrl *gomock.Controller) *MockPathEntryVisitor {
	mock := &MockPathEntryVisitor{ctrl: ctrl}
	mock.recorder = &MockPathEntryVisitorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPathEntryVisitor) EXPECT() *MockPathEntryVisitorMockRecorder {
	return m.recorder
}

// VisitEntry mocks base method
func (m *MockPathEntryVisitor) VisitEntry(arg0 *gitalypb.TreeEntry) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VisitEntry", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VisitEntry indicates an expected call of VisitEntry
func (mr *MockPathEntryVisitorMockRecorder) VisitEntry(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VisitEntry", reflect.TypeOf((*MockPathEntryVisitor)(nil).VisitEntry), arg0)
}
