// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker (interfaces: Tracker)

// Package mock_agent_tracker is a generated GoMock package.
package mock_agent_tracker

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	agent_tracker "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
)

// MockTracker is a mock of Tracker interface.
type MockTracker struct {
	ctrl     *gomock.Controller
	recorder *MockTrackerMockRecorder
}

// MockTrackerMockRecorder is the mock recorder for MockTracker.
type MockTrackerMockRecorder struct {
	mock *MockTracker
}

// NewMockTracker creates a new mock instance.
func NewMockTracker(ctrl *gomock.Controller) *MockTracker {
	mock := &MockTracker{ctrl: ctrl}
	mock.recorder = &MockTrackerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTracker) EXPECT() *MockTrackerMockRecorder {
	return m.recorder
}

// GetConnectionsByAgentId mocks base method.
func (m *MockTracker) GetConnectionsByAgentId(arg0 context.Context, arg1 int64) ([]*agent_tracker.ConnectedAgentInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConnectionsByAgentId", arg0, arg1)
	ret0, _ := ret[0].([]*agent_tracker.ConnectedAgentInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConnectionsByAgentId indicates an expected call of GetConnectionsByAgentId.
func (mr *MockTrackerMockRecorder) GetConnectionsByAgentId(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConnectionsByAgentId", reflect.TypeOf((*MockTracker)(nil).GetConnectionsByAgentId), arg0, arg1)
}

// GetConnectionsByProjectId mocks base method.
func (m *MockTracker) GetConnectionsByProjectId(arg0 context.Context, arg1 int64) ([]*agent_tracker.ConnectedAgentInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConnectionsByProjectId", arg0, arg1)
	ret0, _ := ret[0].([]*agent_tracker.ConnectedAgentInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConnectionsByProjectId indicates an expected call of GetConnectionsByProjectId.
func (mr *MockTrackerMockRecorder) GetConnectionsByProjectId(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConnectionsByProjectId", reflect.TypeOf((*MockTracker)(nil).GetConnectionsByProjectId), arg0, arg1)
}

// RegisterConnection mocks base method.
func (m *MockTracker) RegisterConnection(arg0 context.Context, arg1 *agent_tracker.ConnectedAgentInfo) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterConnection", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// RegisterConnection indicates an expected call of RegisterConnection.
func (mr *MockTrackerMockRecorder) RegisterConnection(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterConnection", reflect.TypeOf((*MockTracker)(nil).RegisterConnection), arg0, arg1)
}

// Run mocks base method.
func (m *MockTracker) Run(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockTrackerMockRecorder) Run(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockTracker)(nil).Run), arg0)
}

// UnregisterConnection mocks base method.
func (m *MockTracker) UnregisterConnection(arg0 context.Context, arg1 *agent_tracker.ConnectedAgentInfo) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnregisterConnection", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// UnregisterConnection indicates an expected call of UnregisterConnection.
func (mr *MockTrackerMockRecorder) UnregisterConnection(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnregisterConnection", reflect.TypeOf((*MockTracker)(nil).UnregisterConnection), arg0, arg1)
}
