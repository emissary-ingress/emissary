// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/lyft/goruntime/snapshot (interfaces: IFace)

// Package mock_snapshot is a generated GoMock package.
package mock_snapshot

import (
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	entry "github.com/lyft/goruntime/snapshot/entry"
)

// MockIFace is a mock of IFace interface
type MockIFace struct {
	ctrl     *gomock.Controller
	recorder *MockIFaceMockRecorder
}

// MockIFaceMockRecorder is the mock recorder for MockIFace
type MockIFaceMockRecorder struct {
	mock *MockIFace
}

// NewMockIFace creates a new mock instance
func NewMockIFace(ctrl *gomock.Controller) *MockIFace {
	mock := &MockIFace{ctrl: ctrl}
	mock.recorder = &MockIFaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockIFace) EXPECT() *MockIFaceMockRecorder {
	return m.recorder
}

// Entries mocks base method
func (m *MockIFace) Entries() map[string]*entry.Entry {
	ret := m.ctrl.Call(m, "Entries")
	ret0, _ := ret[0].(map[string]*entry.Entry)
	return ret0
}

// Entries indicates an expected call of Entries
func (mr *MockIFaceMockRecorder) Entries() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Entries", reflect.TypeOf((*MockIFace)(nil).Entries))
}

// FeatureEnabled mocks base method
func (m *MockIFace) FeatureEnabled(arg0 string, arg1 uint64) bool {
	ret := m.ctrl.Call(m, "FeatureEnabled", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// FeatureEnabled indicates an expected call of FeatureEnabled
func (mr *MockIFaceMockRecorder) FeatureEnabled(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FeatureEnabled", reflect.TypeOf((*MockIFace)(nil).FeatureEnabled), arg0, arg1)
}

// FeatureEnabledForID mocks base method
func (m *MockIFace) FeatureEnabledForID(arg0 string, arg1 uint64, arg2 uint32) bool {
	ret := m.ctrl.Call(m, "FeatureEnabledForID", arg0, arg1, arg2)
	ret0, _ := ret[0].(bool)
	return ret0
}

// FeatureEnabledForID indicates an expected call of FeatureEnabledForID
func (mr *MockIFaceMockRecorder) FeatureEnabledForID(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FeatureEnabledForID", reflect.TypeOf((*MockIFace)(nil).FeatureEnabledForID), arg0, arg1, arg2)
}

// Get mocks base method
func (m *MockIFace) Get(arg0 string) string {
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(string)
	return ret0
}

// Get indicates an expected call of Get
func (mr *MockIFaceMockRecorder) Get(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockIFace)(nil).Get), arg0)
}

// GetInteger mocks base method
func (m *MockIFace) GetInteger(arg0 string, arg1 uint64) uint64 {
	ret := m.ctrl.Call(m, "GetInteger", arg0, arg1)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetInteger indicates an expected call of GetInteger
func (mr *MockIFaceMockRecorder) GetInteger(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInteger", reflect.TypeOf((*MockIFace)(nil).GetInteger), arg0, arg1)
}

// GetModified mocks base method
func (m *MockIFace) GetModified(arg0 string) time.Time {
	ret := m.ctrl.Call(m, "GetModified", arg0)
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// GetModified indicates an expected call of GetModified
func (mr *MockIFaceMockRecorder) GetModified(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModified", reflect.TypeOf((*MockIFace)(nil).GetModified), arg0)
}

// Keys mocks base method
func (m *MockIFace) Keys() []string {
	ret := m.ctrl.Call(m, "Keys")
	ret0, _ := ret[0].([]string)
	return ret0
}

// Keys indicates an expected call of Keys
func (mr *MockIFaceMockRecorder) Keys() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Keys", reflect.TypeOf((*MockIFace)(nil).Keys))
}

// SetEntry mocks base method
func (m *MockIFace) SetEntry(arg0 string, arg1 *entry.Entry) {
	m.ctrl.Call(m, "SetEntry", arg0, arg1)
}

// SetEntry indicates an expected call of SetEntry
func (mr *MockIFaceMockRecorder) SetEntry(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetEntry", reflect.TypeOf((*MockIFace)(nil).SetEntry), arg0, arg1)
}
