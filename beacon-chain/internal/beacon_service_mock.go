// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1 (interfaces: BeaconServiceServer,BeaconService_LatestAttestationServer,BeaconService_WaitForChainStartServer)

// Package internal is a generated GoMock package.
package internal

import (
	context "context"
	reflect "reflect"

	types "github.com/gogo/protobuf/types"
	gomock "github.com/golang/mock/gomock"
	v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	v10 "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	metadata "google.golang.org/grpc/metadata"
)

// MockBeaconServiceServer is a mock of BeaconServiceServer interface
type MockBeaconServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockBeaconServiceServerMockRecorder
}

// MockBeaconServiceServerMockRecorder is the mock recorder for MockBeaconServiceServer
type MockBeaconServiceServerMockRecorder struct {
	mock *MockBeaconServiceServer
}

// NewMockBeaconServiceServer creates a new mock instance
func NewMockBeaconServiceServer(ctrl *gomock.Controller) *MockBeaconServiceServer {
	mock := &MockBeaconServiceServer{ctrl: ctrl}
	mock.recorder = &MockBeaconServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockBeaconServiceServer) EXPECT() *MockBeaconServiceServerMockRecorder {
	return m.recorder
}

// BlockTree mocks base method
func (m *MockBeaconServiceServer) BlockTree(arg0 context.Context, arg1 *types.Empty) (*v10.BlockTreeResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockTree", arg0, arg1)
	ret0, _ := ret[0].(*v10.BlockTreeResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BlockTree indicates an expected call of BlockTree
func (mr *MockBeaconServiceServerMockRecorder) BlockTree(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockTree", reflect.TypeOf((*MockBeaconServiceServer)(nil).BlockTree), arg0, arg1)
}

// BlockTreeBySlots mocks base method
func (m *MockBeaconServiceServer) BlockTreeBySlots(arg0 context.Context, arg1 *v10.TreeBlockSlotRequest) (*v10.BlockTreeResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockTreeBySlots", arg0, arg1)
	ret0, _ := ret[0].(*v10.BlockTreeResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BlockTreeBySlots indicates an expected call of BlockTreeBySlots
func (mr *MockBeaconServiceServerMockRecorder) BlockTreeBySlots(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockTreeBySlots", reflect.TypeOf((*MockBeaconServiceServer)(nil).BlockTreeBySlots), arg0, arg1)
}

// CanonicalHead mocks base method
func (m *MockBeaconServiceServer) CanonicalHead(arg0 context.Context, arg1 *types.Empty) (*v1.BeaconBlock, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CanonicalHead", arg0, arg1)
	ret0, _ := ret[0].(*v1.BeaconBlock)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CanonicalHead indicates an expected call of CanonicalHead
func (mr *MockBeaconServiceServerMockRecorder) CanonicalHead(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CanonicalHead", reflect.TypeOf((*MockBeaconServiceServer)(nil).CanonicalHead), arg0, arg1)
}

// Eth1Data mocks base method
func (m *MockBeaconServiceServer) Eth1Data(arg0 context.Context, arg1 *types.Empty) (*v10.Eth1DataResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Eth1Data", arg0, arg1)
	ret0, _ := ret[0].(*v10.Eth1DataResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Eth1Data indicates an expected call of Eth1Data
func (mr *MockBeaconServiceServerMockRecorder) Eth1Data(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Eth1Data", reflect.TypeOf((*MockBeaconServiceServer)(nil).Eth1Data), arg0, arg1)
}

// ForkData mocks base method
func (m *MockBeaconServiceServer) ForkData(arg0 context.Context, arg1 *types.Empty) (*v1.Fork, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ForkData", arg0, arg1)
	ret0, _ := ret[0].(*v1.Fork)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ForkData indicates an expected call of ForkData
func (mr *MockBeaconServiceServerMockRecorder) ForkData(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForkData", reflect.TypeOf((*MockBeaconServiceServer)(nil).ForkData), arg0, arg1)
}

// LatestAttestation mocks base method
func (m *MockBeaconServiceServer) LatestAttestation(arg0 *types.Empty, arg1 v10.BeaconService_LatestAttestationServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LatestAttestation", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// LatestAttestation indicates an expected call of LatestAttestation
func (mr *MockBeaconServiceServerMockRecorder) LatestAttestation(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LatestAttestation", reflect.TypeOf((*MockBeaconServiceServer)(nil).LatestAttestation), arg0, arg1)
}

// PendingDeposits mocks base method
func (m *MockBeaconServiceServer) PendingDeposits(arg0 context.Context, arg1 *types.Empty) (*v10.PendingDepositsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PendingDeposits", arg0, arg1)
	ret0, _ := ret[0].(*v10.PendingDepositsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PendingDeposits indicates an expected call of PendingDeposits
func (mr *MockBeaconServiceServerMockRecorder) PendingDeposits(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PendingDeposits", reflect.TypeOf((*MockBeaconServiceServer)(nil).PendingDeposits), arg0, arg1)
}

// WaitForChainStart mocks base method
func (m *MockBeaconServiceServer) WaitForChainStart(arg0 *types.Empty, arg1 v10.BeaconService_WaitForChainStartServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForChainStart", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForChainStart indicates an expected call of WaitForChainStart
func (mr *MockBeaconServiceServerMockRecorder) WaitForChainStart(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForChainStart", reflect.TypeOf((*MockBeaconServiceServer)(nil).WaitForChainStart), arg0, arg1)
}

// MockBeaconService_LatestAttestationServer is a mock of BeaconService_LatestAttestationServer interface
type MockBeaconService_LatestAttestationServer struct {
	ctrl     *gomock.Controller
	recorder *MockBeaconService_LatestAttestationServerMockRecorder
}

// MockBeaconService_LatestAttestationServerMockRecorder is the mock recorder for MockBeaconService_LatestAttestationServer
type MockBeaconService_LatestAttestationServerMockRecorder struct {
	mock *MockBeaconService_LatestAttestationServer
}

// NewMockBeaconService_LatestAttestationServer creates a new mock instance
func NewMockBeaconService_LatestAttestationServer(ctrl *gomock.Controller) *MockBeaconService_LatestAttestationServer {
	mock := &MockBeaconService_LatestAttestationServer{ctrl: ctrl}
	mock.recorder = &MockBeaconService_LatestAttestationServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockBeaconService_LatestAttestationServer) EXPECT() *MockBeaconService_LatestAttestationServerMockRecorder {
	return m.recorder
}

// Context mocks base method
func (m *MockBeaconService_LatestAttestationServer) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context
func (mr *MockBeaconService_LatestAttestationServerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockBeaconService_LatestAttestationServer)(nil).Context))
}

// RecvMsg mocks base method
func (m *MockBeaconService_LatestAttestationServer) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg
func (mr *MockBeaconService_LatestAttestationServerMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockBeaconService_LatestAttestationServer)(nil).RecvMsg), arg0)
}

// Send mocks base method
func (m *MockBeaconService_LatestAttestationServer) Send(arg0 *v1.Attestation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send
func (mr *MockBeaconService_LatestAttestationServerMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockBeaconService_LatestAttestationServer)(nil).Send), arg0)
}

// SendHeader mocks base method
func (m *MockBeaconService_LatestAttestationServer) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader
func (mr *MockBeaconService_LatestAttestationServerMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockBeaconService_LatestAttestationServer)(nil).SendHeader), arg0)
}

// SendMsg mocks base method
func (m *MockBeaconService_LatestAttestationServer) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg
func (mr *MockBeaconService_LatestAttestationServerMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockBeaconService_LatestAttestationServer)(nil).SendMsg), arg0)
}

// SetHeader mocks base method
func (m *MockBeaconService_LatestAttestationServer) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader
func (mr *MockBeaconService_LatestAttestationServerMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockBeaconService_LatestAttestationServer)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method
func (m *MockBeaconService_LatestAttestationServer) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer
func (mr *MockBeaconService_LatestAttestationServerMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockBeaconService_LatestAttestationServer)(nil).SetTrailer), arg0)
}

// MockBeaconService_WaitForChainStartServer is a mock of BeaconService_WaitForChainStartServer interface
type MockBeaconService_WaitForChainStartServer struct {
	ctrl     *gomock.Controller
	recorder *MockBeaconService_WaitForChainStartServerMockRecorder
}

// MockBeaconService_WaitForChainStartServerMockRecorder is the mock recorder for MockBeaconService_WaitForChainStartServer
type MockBeaconService_WaitForChainStartServerMockRecorder struct {
	mock *MockBeaconService_WaitForChainStartServer
}

// NewMockBeaconService_WaitForChainStartServer creates a new mock instance
func NewMockBeaconService_WaitForChainStartServer(ctrl *gomock.Controller) *MockBeaconService_WaitForChainStartServer {
	mock := &MockBeaconService_WaitForChainStartServer{ctrl: ctrl}
	mock.recorder = &MockBeaconService_WaitForChainStartServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockBeaconService_WaitForChainStartServer) EXPECT() *MockBeaconService_WaitForChainStartServerMockRecorder {
	return m.recorder
}

// Context mocks base method
func (m *MockBeaconService_WaitForChainStartServer) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context
func (mr *MockBeaconService_WaitForChainStartServerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockBeaconService_WaitForChainStartServer)(nil).Context))
}

// RecvMsg mocks base method
func (m *MockBeaconService_WaitForChainStartServer) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg
func (mr *MockBeaconService_WaitForChainStartServerMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockBeaconService_WaitForChainStartServer)(nil).RecvMsg), arg0)
}

// Send mocks base method
func (m *MockBeaconService_WaitForChainStartServer) Send(arg0 *v10.ChainStartResponse) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send
func (mr *MockBeaconService_WaitForChainStartServerMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockBeaconService_WaitForChainStartServer)(nil).Send), arg0)
}

// SendHeader mocks base method
func (m *MockBeaconService_WaitForChainStartServer) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader
func (mr *MockBeaconService_WaitForChainStartServerMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockBeaconService_WaitForChainStartServer)(nil).SendHeader), arg0)
}

// SendMsg mocks base method
func (m *MockBeaconService_WaitForChainStartServer) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg
func (mr *MockBeaconService_WaitForChainStartServerMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockBeaconService_WaitForChainStartServer)(nil).SendMsg), arg0)
}

// SetHeader mocks base method
func (m *MockBeaconService_WaitForChainStartServer) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader
func (mr *MockBeaconService_WaitForChainStartServerMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockBeaconService_WaitForChainStartServer)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method
func (m *MockBeaconService_WaitForChainStartServer) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer
func (mr *MockBeaconService_WaitForChainStartServerMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockBeaconService_WaitForChainStartServer)(nil).SetTrailer), arg0)
}
