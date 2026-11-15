package budget

import (
	reflect "reflect"
	time "time"

	gomock "go.uber.org/mock/gomock"
	unix "golang.org/x/sys/unix"
)

type MockadmissionHostAdapter struct {
	ctrl     *gomock.Controller
	recorder *MockadmissionHostAdapterMockRecorder
	isgomock struct{}
}

type MockadmissionHostAdapterMockRecorder struct {
	mock *MockadmissionHostAdapter
}

func NewMockadmissionHostAdapter(ctrl *gomock.Controller) *MockadmissionHostAdapter {
	mock := &MockadmissionHostAdapter{ctrl: ctrl}
	mock.recorder = &MockadmissionHostAdapterMockRecorder{mock}
	return mock
}

func (m *MockadmissionHostAdapter) EXPECT() *MockadmissionHostAdapterMockRecorder {
	return m.recorder
}

func (m *MockadmissionHostAdapter) Now() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Now")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

func (mr *MockadmissionHostAdapterMockRecorder) Now() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Now", reflect.TypeOf((*MockadmissionHostAdapter)(nil).Now))
}

func (m *MockadmissionHostAdapter) Statfs(arg0 int) (unix.Statfs_t, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Statfs", arg0)
	ret0, _ := ret[0].(unix.Statfs_t)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockadmissionHostAdapterMockRecorder) Statfs(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Statfs", reflect.TypeOf((*MockadmissionHostAdapter)(nil).Statfs), arg0)
}

func (m *MockadmissionHostAdapter) SysctlRaw(arg0 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SysctlRaw", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockadmissionHostAdapterMockRecorder) SysctlRaw(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SysctlRaw", reflect.TypeOf((*MockadmissionHostAdapter)(nil).SysctlRaw), arg0)
}
