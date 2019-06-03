// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/m3db/m3db-operator/pkg/k8sops/types.go

// Copyright (c) 2019 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package k8sops is a generated GoMock package.
package k8sops

import (
	"reflect"

	"github.com/m3db/m3db-operator/pkg/apis/m3dboperator/v1alpha1"

	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	v10 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// MockK8sops is a mock of K8sops interface
type MockK8sops struct {
	ctrl     *gomock.Controller
	recorder *MockK8sopsMockRecorder
}

// MockK8sopsMockRecorder is the mock recorder for MockK8sops
type MockK8sopsMockRecorder struct {
	mock *MockK8sops
}

// NewMockK8sops creates a new mock instance
func NewMockK8sops(ctrl *gomock.Controller) *MockK8sops {
	mock := &MockK8sops{ctrl: ctrl}
	mock.recorder = &MockK8sopsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockK8sops) EXPECT() *MockK8sopsMockRecorder {
	return m.recorder
}

// CreateOrUpdateCRD mocks base method
func (m *MockK8sops) CreateOrUpdateCRD(name string, enableValidation bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrUpdateCRD", name, enableValidation)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateOrUpdateCRD indicates an expected call of CreateOrUpdateCRD
func (mr *MockK8sopsMockRecorder) CreateOrUpdateCRD(name, enableValidation interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrUpdateCRD", reflect.TypeOf((*MockK8sops)(nil).CreateOrUpdateCRD), name, enableValidation)
}

// GetService mocks base method
func (m *MockK8sops) GetService(cluster *v1alpha1.M3DBCluster, name string) (*v1.Service, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetService", cluster, name)
	ret0, _ := ret[0].(*v1.Service)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetService indicates an expected call of GetService
func (mr *MockK8sopsMockRecorder) GetService(cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetService", reflect.TypeOf((*MockK8sops)(nil).GetService), cluster, name)
}

// DeleteService mocks base method
func (m *MockK8sops) DeleteService(cluster *v1alpha1.M3DBCluster, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteService", cluster, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteService indicates an expected call of DeleteService
func (mr *MockK8sopsMockRecorder) DeleteService(cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteService", reflect.TypeOf((*MockK8sops)(nil).DeleteService), cluster, name)
}

// EnsureService mocks base method
func (m *MockK8sops) EnsureService(cluster *v1alpha1.M3DBCluster, svc *v1.Service) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureService", cluster, svc)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnsureService indicates an expected call of EnsureService
func (mr *MockK8sopsMockRecorder) EnsureService(cluster, svc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureService", reflect.TypeOf((*MockK8sops)(nil).EnsureService), cluster, svc)
}

// Events mocks base method
func (m *MockK8sops) Events(namespace string) v10.EventInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Events", namespace)
	ret0, _ := ret[0].(v10.EventInterface)
	return ret0
}

// Events indicates an expected call of Events
func (mr *MockK8sopsMockRecorder) Events(namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Events", reflect.TypeOf((*MockK8sops)(nil).Events), namespace)
}
