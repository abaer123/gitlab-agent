// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modagent (interfaces: API,Factory,Module)

// Package mock_modagent is a generated GoMock package.
package mock_modagent

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	modagent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modagent"
	agentcfg "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
)

// MockAPI is a mock of API interface
type MockAPI struct {
	ctrl     *gomock.Controller
	recorder *MockAPIMockRecorder
}

// MockAPIMockRecorder is the mock recorder for MockAPI
type MockAPIMockRecorder struct {
	mock *MockAPI
}

// NewMockAPI creates a new mock instance
func NewMockAPI(ctrl *gomock.Controller) *MockAPI {
	mock := &MockAPI{ctrl: ctrl}
	mock.recorder = &MockAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAPI) EXPECT() *MockAPIMockRecorder {
	return m.recorder
}

// MockFactory is a mock of Factory interface
type MockFactory struct {
	ctrl     *gomock.Controller
	recorder *MockFactoryMockRecorder
}

// MockFactoryMockRecorder is the mock recorder for MockFactory
type MockFactoryMockRecorder struct {
	mock *MockFactory
}

// NewMockFactory creates a new mock instance
func NewMockFactory(ctrl *gomock.Controller) *MockFactory {
	mock := &MockFactory{ctrl: ctrl}
	mock.recorder = &MockFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFactory) EXPECT() *MockFactoryMockRecorder {
	return m.recorder
}

// New mocks base method
func (m *MockFactory) New(arg0 *modagent.Config) modagent.Module {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "New", arg0)
	ret0, _ := ret[0].(modagent.Module)
	return ret0
}

// New indicates an expected call of New
func (mr *MockFactoryMockRecorder) New(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "New", reflect.TypeOf((*MockFactory)(nil).New), arg0)
}

// MockModule is a mock of Module interface
type MockModule struct {
	ctrl     *gomock.Controller
	recorder *MockModuleMockRecorder
}

// MockModuleMockRecorder is the mock recorder for MockModule
type MockModuleMockRecorder struct {
	mock *MockModule
}

// NewMockModule creates a new mock instance
func NewMockModule(ctrl *gomock.Controller) *MockModule {
	mock := &MockModule{ctrl: ctrl}
	mock.recorder = &MockModuleMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockModule) EXPECT() *MockModuleMockRecorder {
	return m.recorder
}

// DefaultAndValidateConfiguration mocks base method
func (m *MockModule) DefaultAndValidateConfiguration(arg0 *agentcfg.AgentConfiguration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DefaultAndValidateConfiguration", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DefaultAndValidateConfiguration indicates an expected call of DefaultAndValidateConfiguration
func (mr *MockModuleMockRecorder) DefaultAndValidateConfiguration(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DefaultAndValidateConfiguration", reflect.TypeOf((*MockModule)(nil).DefaultAndValidateConfiguration), arg0)
}

// Name mocks base method
func (m *MockModule) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (mr *MockModuleMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockModule)(nil).Name))
}

// Run mocks base method
func (m *MockModule) Run(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run
func (mr *MockModuleMockRecorder) Run(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockModule)(nil).Run), arg0)
}

// SetConfiguration mocks base method
func (m *MockModule) SetConfiguration(arg0 *agentcfg.AgentConfiguration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetConfiguration", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetConfiguration indicates an expected call of SetConfiguration
func (mr *MockModuleMockRecorder) SetConfiguration(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetConfiguration", reflect.TypeOf((*MockModule)(nil).SetConfiguration), arg0)
}
