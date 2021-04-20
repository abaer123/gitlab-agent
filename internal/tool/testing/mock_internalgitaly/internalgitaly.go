// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly (interfaces: PoolInterface,FetchVisitor,PathEntryVisitor,FileVisitor,PathFetcherInterface,PollerInterface)

// Package mock_internalgitaly is a generated GoMock package.
package mock_internalgitaly

import (
	"context"
	"reflect"

	"github.com/golang/mock/gomock"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// MockPoolInterface is a mock of PoolInterface interface.
type MockPoolInterface struct {
	ctrl     *gomock.Controller
	recorder *MockPoolInterfaceMockRecorder
}

// MockPoolInterfaceMockRecorder is the mock recorder for MockPoolInterface.
type MockPoolInterfaceMockRecorder struct {
	mock *MockPoolInterface
}

// NewMockPoolInterface creates a new mock instance.
func NewMockPoolInterface(ctrl *gomock.Controller) *MockPoolInterface {
	mock := &MockPoolInterface{ctrl: ctrl}
	mock.recorder = &MockPoolInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPoolInterface) EXPECT() *MockPoolInterfaceMockRecorder {
	return m.recorder
}

// PathFetcher mocks base method.
func (m *MockPoolInterface) PathFetcher(arg0 context.Context, arg1 *api.GitalyInfo) (gitaly.PathFetcherInterface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PathFetcher", arg0, arg1)
	ret0, _ := ret[0].(gitaly.PathFetcherInterface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PathFetcher indicates an expected call of PathFetcher.
func (mr *MockPoolInterfaceMockRecorder) PathFetcher(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PathFetcher", reflect.TypeOf((*MockPoolInterface)(nil).PathFetcher), arg0, arg1)
}

// Poller mocks base method.
func (m *MockPoolInterface) Poller(arg0 context.Context, arg1 *api.GitalyInfo) (gitaly.PollerInterface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Poller", arg0, arg1)
	ret0, _ := ret[0].(gitaly.PollerInterface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Poller indicates an expected call of Poller.
func (mr *MockPoolInterfaceMockRecorder) Poller(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Poller", reflect.TypeOf((*MockPoolInterface)(nil).Poller), arg0, arg1)
}

// MockFetchVisitor is a mock of FetchVisitor interface.
type MockFetchVisitor struct {
	ctrl     *gomock.Controller
	recorder *MockFetchVisitorMockRecorder
}

// MockFetchVisitorMockRecorder is the mock recorder for MockFetchVisitor.
type MockFetchVisitorMockRecorder struct {
	mock *MockFetchVisitor
}

// NewMockFetchVisitor creates a new mock instance.
func NewMockFetchVisitor(ctrl *gomock.Controller) *MockFetchVisitor {
	mock := &MockFetchVisitor{ctrl: ctrl}
	mock.recorder = &MockFetchVisitorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFetchVisitor) EXPECT() *MockFetchVisitorMockRecorder {
	return m.recorder
}

// Entry mocks base method.
func (m *MockFetchVisitor) Entry(arg0 *gitalypb.TreeEntry) (bool, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Entry", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Entry indicates an expected call of Entry.
func (mr *MockFetchVisitorMockRecorder) Entry(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Entry", reflect.TypeOf((*MockFetchVisitor)(nil).Entry), arg0)
}

// EntryDone mocks base method.
func (m *MockFetchVisitor) EntryDone(arg0 *gitalypb.TreeEntry, arg1 error) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "EntryDone", arg0, arg1)
}

// EntryDone indicates an expected call of EntryDone.
func (mr *MockFetchVisitorMockRecorder) EntryDone(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EntryDone", reflect.TypeOf((*MockFetchVisitor)(nil).EntryDone), arg0, arg1)
}

// StreamChunk mocks base method.
func (m *MockFetchVisitor) StreamChunk(arg0, arg1 []byte) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StreamChunk", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StreamChunk indicates an expected call of StreamChunk.
func (mr *MockFetchVisitorMockRecorder) StreamChunk(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StreamChunk", reflect.TypeOf((*MockFetchVisitor)(nil).StreamChunk), arg0, arg1)
}

// MockPathEntryVisitor is a mock of PathEntryVisitor interface.
type MockPathEntryVisitor struct {
	ctrl     *gomock.Controller
	recorder *MockPathEntryVisitorMockRecorder
}

// MockPathEntryVisitorMockRecorder is the mock recorder for MockPathEntryVisitor.
type MockPathEntryVisitorMockRecorder struct {
	mock *MockPathEntryVisitor
}

// NewMockPathEntryVisitor creates a new mock instance.
func NewMockPathEntryVisitor(ctrl *gomock.Controller) *MockPathEntryVisitor {
	mock := &MockPathEntryVisitor{ctrl: ctrl}
	mock.recorder = &MockPathEntryVisitorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPathEntryVisitor) EXPECT() *MockPathEntryVisitorMockRecorder {
	return m.recorder
}

// Entry mocks base method.
func (m *MockPathEntryVisitor) Entry(arg0 *gitalypb.TreeEntry) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Entry", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Entry indicates an expected call of Entry.
func (mr *MockPathEntryVisitorMockRecorder) Entry(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Entry", reflect.TypeOf((*MockPathEntryVisitor)(nil).Entry), arg0)
}

// MockFileVisitor is a mock of FileVisitor interface.
type MockFileVisitor struct {
	ctrl     *gomock.Controller
	recorder *MockFileVisitorMockRecorder
}

// MockFileVisitorMockRecorder is the mock recorder for MockFileVisitor.
type MockFileVisitorMockRecorder struct {
	mock *MockFileVisitor
}

// NewMockFileVisitor creates a new mock instance.
func NewMockFileVisitor(ctrl *gomock.Controller) *MockFileVisitor {
	mock := &MockFileVisitor{ctrl: ctrl}
	mock.recorder = &MockFileVisitorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFileVisitor) EXPECT() *MockFileVisitorMockRecorder {
	return m.recorder
}

// Chunk mocks base method.
func (m *MockFileVisitor) Chunk(arg0 []byte) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chunk", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Chunk indicates an expected call of Chunk.
func (mr *MockFileVisitorMockRecorder) Chunk(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chunk", reflect.TypeOf((*MockFileVisitor)(nil).Chunk), arg0)
}

// MockPathFetcherInterface is a mock of PathFetcherInterface interface.
type MockPathFetcherInterface struct {
	ctrl     *gomock.Controller
	recorder *MockPathFetcherInterfaceMockRecorder
}

// MockPathFetcherInterfaceMockRecorder is the mock recorder for MockPathFetcherInterface.
type MockPathFetcherInterfaceMockRecorder struct {
	mock *MockPathFetcherInterface
}

// NewMockPathFetcherInterface creates a new mock instance.
func NewMockPathFetcherInterface(ctrl *gomock.Controller) *MockPathFetcherInterface {
	mock := &MockPathFetcherInterface{ctrl: ctrl}
	mock.recorder = &MockPathFetcherInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPathFetcherInterface) EXPECT() *MockPathFetcherInterfaceMockRecorder {
	return m.recorder
}

// FetchFile mocks base method.
func (m *MockPathFetcherInterface) FetchFile(arg0 context.Context, arg1 *gitalypb.Repository, arg2, arg3 []byte, arg4 int64) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchFile", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchFile indicates an expected call of FetchFile.
func (mr *MockPathFetcherInterfaceMockRecorder) FetchFile(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchFile", reflect.TypeOf((*MockPathFetcherInterface)(nil).FetchFile), arg0, arg1, arg2, arg3, arg4)
}

// StreamFile mocks base method.
func (m *MockPathFetcherInterface) StreamFile(arg0 context.Context, arg1 *gitalypb.Repository, arg2, arg3 []byte, arg4 int64, arg5 gitaly.FileVisitor) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StreamFile", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(error)
	return ret0
}

// StreamFile indicates an expected call of StreamFile.
func (mr *MockPathFetcherInterfaceMockRecorder) StreamFile(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StreamFile", reflect.TypeOf((*MockPathFetcherInterface)(nil).StreamFile), arg0, arg1, arg2, arg3, arg4, arg5)
}

// Visit mocks base method.
func (m *MockPathFetcherInterface) Visit(arg0 context.Context, arg1 *gitalypb.Repository, arg2, arg3 []byte, arg4 bool, arg5 gitaly.FetchVisitor) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Visit", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(error)
	return ret0
}

// Visit indicates an expected call of Visit.
func (mr *MockPathFetcherInterfaceMockRecorder) Visit(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Visit", reflect.TypeOf((*MockPathFetcherInterface)(nil).Visit), arg0, arg1, arg2, arg3, arg4, arg5)
}

// MockPollerInterface is a mock of PollerInterface interface.
type MockPollerInterface struct {
	ctrl     *gomock.Controller
	recorder *MockPollerInterfaceMockRecorder
}

// MockPollerInterfaceMockRecorder is the mock recorder for MockPollerInterface.
type MockPollerInterfaceMockRecorder struct {
	mock *MockPollerInterface
}

// NewMockPollerInterface creates a new mock instance.
func NewMockPollerInterface(ctrl *gomock.Controller) *MockPollerInterface {
	mock := &MockPollerInterface{ctrl: ctrl}
	mock.recorder = &MockPollerInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPollerInterface) EXPECT() *MockPollerInterfaceMockRecorder {
	return m.recorder
}

// Poll mocks base method.
func (m *MockPollerInterface) Poll(arg0 context.Context, arg1 *gitalypb.Repository, arg2, arg3 string) (*gitaly.PollInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Poll", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*gitaly.PollInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Poll indicates an expected call of Poll.
func (mr *MockPollerInterfaceMockRecorder) Poll(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Poll", reflect.TypeOf((*MockPollerInterface)(nil).Poll), arg0, arg1, arg2, arg3)
}
