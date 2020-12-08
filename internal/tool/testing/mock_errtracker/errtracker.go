// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/gitlab-org/labkit/errortracking (interfaces: Tracker)

// Package mock_errtracker is a generated GoMock package.
package mock_errtracker

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	errortracking "gitlab.com/gitlab-org/labkit/errortracking"
)

// MockTracker is a mock of Tracker interface
type MockTracker struct {
	ctrl     *gomock.Controller
	recorder *MockTrackerMockRecorder
}

// MockTrackerMockRecorder is the mock recorder for MockTracker
type MockTrackerMockRecorder struct {
	mock *MockTracker
}

// NewMockTracker creates a new mock instance
func NewMockTracker(ctrl *gomock.Controller) *MockTracker {
	mock := &MockTracker{ctrl: ctrl}
	mock.recorder = &MockTrackerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTracker) EXPECT() *MockTrackerMockRecorder {
	return m.recorder
}

// Capture mocks base method
func (m *MockTracker) Capture(arg0 error, arg1 ...errortracking.CaptureOption) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Capture", varargs...)
}

// Capture indicates an expected call of Capture
func (mr *MockTrackerMockRecorder) Capture(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Capture", reflect.TypeOf((*MockTracker)(nil).Capture), varargs...)
}