// Code generated by MockGen. DO NOT EDIT.
// Source: rsync.go

// Package rsync is a generated GoMock package.
package rsync

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	args "github.com/release-engineering/exodus-rsync/internal/args"
	conf "github.com/release-engineering/exodus-rsync/internal/conf"
)

// MockInterface is a mock of Interface interface.
type MockInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceMockRecorder
}

// MockInterfaceMockRecorder is the mock recorder for MockInterface.
type MockInterfaceMockRecorder struct {
	mock *MockInterface
}

// NewMockInterface creates a new mock instance.
func NewMockInterface(ctrl *gomock.Controller) *MockInterface {
	mock := &MockInterface{ctrl: ctrl}
	mock.recorder = &MockInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterface) EXPECT() *MockInterfaceMockRecorder {
	return m.recorder
}

// Exec mocks base method.
func (m *MockInterface) Exec(arg0 context.Context, arg1 conf.Config, arg2 args.Config) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exec", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Exec indicates an expected call of Exec.
func (mr *MockInterfaceMockRecorder) Exec(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exec", reflect.TypeOf((*MockInterface)(nil).Exec), arg0, arg1, arg2)
}