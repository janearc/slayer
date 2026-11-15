package omnipositor

import (
	context "context"
	gomock "go.uber.org/mock/gomock"
	reflect "reflect"
)

type MockLightStage struct {
	ctrl     *gomock.Controller
	recorder *MockLightStageMockRecorder
	isgomock struct{}
}

type MockLightStageMockRecorder struct{ mock *MockLightStage }

func NewMockLightStage(ctrl *gomock.Controller) *MockLightStage {
	mock := &MockLightStage{ctrl: ctrl}
	mock.recorder = &MockLightStageMockRecorder{mock}
	return mock
}

func (m *MockLightStage) EXPECT() *MockLightStageMockRecorder { return m.recorder }

func (m *MockLightStage) Light(arg0 context.Context, arg1 LightRequest) (LightResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Light", arg0, arg1)
	ret0, _ := ret[0].(LightResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockLightStageMockRecorder) Light(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Light", reflect.TypeOf((*MockLightStage)(nil).Light), arg0, arg1)
}
