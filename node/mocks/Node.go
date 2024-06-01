// Code generated by mockery v2.42.0. DO NOT EDIT.

package mocks

import (
	context "context"

	entries "github.com/r-moraru/modular-raft/proto/entries"
	mock "github.com/stretchr/testify/mock"

	node "github.com/r-moraru/modular-raft/node"
)

// Node is an autogenerated mock type for the Node type
type Node struct {
	mock.Mock
}

type Node_Expecter struct {
	mock *mock.Mock
}

func (_m *Node) EXPECT() *Node_Expecter {
	return &Node_Expecter{mock: &_m.Mock}
}

// AppendEntry provides a mock function with given fields: entry
func (_m *Node) AppendEntry(entry *entries.LogEntry) error {
	ret := _m.Called(entry)

	if len(ret) == 0 {
		panic("no return value specified for AppendEntry")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*entries.LogEntry) error); ok {
		r0 = rf(entry)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Node_AppendEntry_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AppendEntry'
type Node_AppendEntry_Call struct {
	*mock.Call
}

// AppendEntry is a helper method to define mock.On call
//   - entry *entries.LogEntry
func (_e *Node_Expecter) AppendEntry(entry interface{}) *Node_AppendEntry_Call {
	return &Node_AppendEntry_Call{Call: _e.mock.On("AppendEntry", entry)}
}

func (_c *Node_AppendEntry_Call) Run(run func(entry *entries.LogEntry)) *Node_AppendEntry_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*entries.LogEntry))
	})
	return _c
}

func (_c *Node_AppendEntry_Call) Return(_a0 error) *Node_AppendEntry_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_AppendEntry_Call) RunAndReturn(run func(*entries.LogEntry) error) *Node_AppendEntry_Call {
	_c.Call.Return(run)
	return _c
}

// ClearVotedFor provides a mock function with given fields:
func (_m *Node) ClearVotedFor() {
	_m.Called()
}

// Node_ClearVotedFor_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ClearVotedFor'
type Node_ClearVotedFor_Call struct {
	*mock.Call
}

// ClearVotedFor is a helper method to define mock.On call
func (_e *Node_Expecter) ClearVotedFor() *Node_ClearVotedFor_Call {
	return &Node_ClearVotedFor_Call{Call: _e.mock.On("ClearVotedFor")}
}

func (_c *Node_ClearVotedFor_Call) Run(run func()) *Node_ClearVotedFor_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_ClearVotedFor_Call) Return() *Node_ClearVotedFor_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_ClearVotedFor_Call) RunAndReturn(run func()) *Node_ClearVotedFor_Call {
	_c.Call.Return(run)
	return _c
}

// GetCommitIndex provides a mock function with given fields:
func (_m *Node) GetCommitIndex() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetCommitIndex")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// Node_GetCommitIndex_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCommitIndex'
type Node_GetCommitIndex_Call struct {
	*mock.Call
}

// GetCommitIndex is a helper method to define mock.On call
func (_e *Node_Expecter) GetCommitIndex() *Node_GetCommitIndex_Call {
	return &Node_GetCommitIndex_Call{Call: _e.mock.On("GetCommitIndex")}
}

func (_c *Node_GetCommitIndex_Call) Run(run func()) *Node_GetCommitIndex_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_GetCommitIndex_Call) Return(_a0 uint64) *Node_GetCommitIndex_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_GetCommitIndex_Call) RunAndReturn(run func() uint64) *Node_GetCommitIndex_Call {
	_c.Call.Return(run)
	return _c
}

// GetCurrentLeaderID provides a mock function with given fields:
func (_m *Node) GetCurrentLeaderID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetCurrentLeaderID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Node_GetCurrentLeaderID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCurrentLeaderID'
type Node_GetCurrentLeaderID_Call struct {
	*mock.Call
}

// GetCurrentLeaderID is a helper method to define mock.On call
func (_e *Node_Expecter) GetCurrentLeaderID() *Node_GetCurrentLeaderID_Call {
	return &Node_GetCurrentLeaderID_Call{Call: _e.mock.On("GetCurrentLeaderID")}
}

func (_c *Node_GetCurrentLeaderID_Call) Run(run func()) *Node_GetCurrentLeaderID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_GetCurrentLeaderID_Call) Return(_a0 string) *Node_GetCurrentLeaderID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_GetCurrentLeaderID_Call) RunAndReturn(run func() string) *Node_GetCurrentLeaderID_Call {
	_c.Call.Return(run)
	return _c
}

// GetCurrentTerm provides a mock function with given fields:
func (_m *Node) GetCurrentTerm() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetCurrentTerm")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// Node_GetCurrentTerm_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCurrentTerm'
type Node_GetCurrentTerm_Call struct {
	*mock.Call
}

// GetCurrentTerm is a helper method to define mock.On call
func (_e *Node_Expecter) GetCurrentTerm() *Node_GetCurrentTerm_Call {
	return &Node_GetCurrentTerm_Call{Call: _e.mock.On("GetCurrentTerm")}
}

func (_c *Node_GetCurrentTerm_Call) Run(run func()) *Node_GetCurrentTerm_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_GetCurrentTerm_Call) Return(_a0 uint64) *Node_GetCurrentTerm_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_GetCurrentTerm_Call) RunAndReturn(run func() uint64) *Node_GetCurrentTerm_Call {
	_c.Call.Return(run)
	return _c
}

// GetLastLogIndex provides a mock function with given fields:
func (_m *Node) GetLastLogIndex() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLastLogIndex")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// Node_GetLastLogIndex_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLastLogIndex'
type Node_GetLastLogIndex_Call struct {
	*mock.Call
}

// GetLastLogIndex is a helper method to define mock.On call
func (_e *Node_Expecter) GetLastLogIndex() *Node_GetLastLogIndex_Call {
	return &Node_GetLastLogIndex_Call{Call: _e.mock.On("GetLastLogIndex")}
}

func (_c *Node_GetLastLogIndex_Call) Run(run func()) *Node_GetLastLogIndex_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_GetLastLogIndex_Call) Return(_a0 uint64) *Node_GetLastLogIndex_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_GetLastLogIndex_Call) RunAndReturn(run func() uint64) *Node_GetLastLogIndex_Call {
	_c.Call.Return(run)
	return _c
}

// GetLogLength provides a mock function with given fields:
func (_m *Node) GetLogLength() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLogLength")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// Node_GetLogLength_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLogLength'
type Node_GetLogLength_Call struct {
	*mock.Call
}

// GetLogLength is a helper method to define mock.On call
func (_e *Node_Expecter) GetLogLength() *Node_GetLogLength_Call {
	return &Node_GetLogLength_Call{Call: _e.mock.On("GetLogLength")}
}

func (_c *Node_GetLogLength_Call) Run(run func()) *Node_GetLogLength_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_GetLogLength_Call) Return(_a0 uint64) *Node_GetLogLength_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_GetLogLength_Call) RunAndReturn(run func() uint64) *Node_GetLogLength_Call {
	_c.Call.Return(run)
	return _c
}

// GetState provides a mock function with given fields:
func (_m *Node) GetState() node.State {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetState")
	}

	var r0 node.State
	if rf, ok := ret.Get(0).(func() node.State); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(node.State)
	}

	return r0
}

// Node_GetState_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetState'
type Node_GetState_Call struct {
	*mock.Call
}

// GetState is a helper method to define mock.On call
func (_e *Node_Expecter) GetState() *Node_GetState_Call {
	return &Node_GetState_Call{Call: _e.mock.On("GetState")}
}

func (_c *Node_GetState_Call) Run(run func()) *Node_GetState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_GetState_Call) Return(_a0 node.State) *Node_GetState_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_GetState_Call) RunAndReturn(run func() node.State) *Node_GetState_Call {
	_c.Call.Return(run)
	return _c
}

// GetTermAtIndex provides a mock function with given fields: index
func (_m *Node) GetTermAtIndex(index uint64) (uint64, error) {
	ret := _m.Called(index)

	if len(ret) == 0 {
		panic("no return value specified for GetTermAtIndex")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (uint64, error)); ok {
		return rf(index)
	}
	if rf, ok := ret.Get(0).(func(uint64) uint64); ok {
		r0 = rf(index)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(index)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Node_GetTermAtIndex_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetTermAtIndex'
type Node_GetTermAtIndex_Call struct {
	*mock.Call
}

// GetTermAtIndex is a helper method to define mock.On call
//   - index uint64
func (_e *Node_Expecter) GetTermAtIndex(index interface{}) *Node_GetTermAtIndex_Call {
	return &Node_GetTermAtIndex_Call{Call: _e.mock.On("GetTermAtIndex", index)}
}

func (_c *Node_GetTermAtIndex_Call) Run(run func(index uint64)) *Node_GetTermAtIndex_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *Node_GetTermAtIndex_Call) Return(_a0 uint64, _a1 error) *Node_GetTermAtIndex_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Node_GetTermAtIndex_Call) RunAndReturn(run func(uint64) (uint64, error)) *Node_GetTermAtIndex_Call {
	_c.Call.Return(run)
	return _c
}

// GetVotedFor provides a mock function with given fields:
func (_m *Node) GetVotedFor() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetVotedFor")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Node_GetVotedFor_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetVotedFor'
type Node_GetVotedFor_Call struct {
	*mock.Call
}

// GetVotedFor is a helper method to define mock.On call
func (_e *Node_Expecter) GetVotedFor() *Node_GetVotedFor_Call {
	return &Node_GetVotedFor_Call{Call: _e.mock.On("GetVotedFor")}
}

func (_c *Node_GetVotedFor_Call) Run(run func()) *Node_GetVotedFor_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_GetVotedFor_Call) Return(_a0 string) *Node_GetVotedFor_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_GetVotedFor_Call) RunAndReturn(run func() string) *Node_GetVotedFor_Call {
	_c.Call.Return(run)
	return _c
}

// ResetTimer provides a mock function with given fields:
func (_m *Node) ResetTimer() {
	_m.Called()
}

// Node_ResetTimer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ResetTimer'
type Node_ResetTimer_Call struct {
	*mock.Call
}

// ResetTimer is a helper method to define mock.On call
func (_e *Node_Expecter) ResetTimer() *Node_ResetTimer_Call {
	return &Node_ResetTimer_Call{Call: _e.mock.On("ResetTimer")}
}

func (_c *Node_ResetTimer_Call) Run(run func()) *Node_ResetTimer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_ResetTimer_Call) Return() *Node_ResetTimer_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_ResetTimer_Call) RunAndReturn(run func()) *Node_ResetTimer_Call {
	_c.Call.Return(run)
	return _c
}

// Run provides a mock function with given fields: ctx
func (_m *Node) Run(ctx context.Context) {
	_m.Called(ctx)
}

// Node_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type Node_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Node_Expecter) Run(ctx interface{}) *Node_Run_Call {
	return &Node_Run_Call{Call: _e.mock.On("Run", ctx)}
}

func (_c *Node_Run_Call) Run(run func(ctx context.Context)) *Node_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Node_Run_Call) Return() *Node_Run_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_Run_Call) RunAndReturn(run func(context.Context)) *Node_Run_Call {
	_c.Call.Return(run)
	return _c
}

// SetCommitIndex provides a mock function with given fields: commitIndex
func (_m *Node) SetCommitIndex(commitIndex uint64) {
	_m.Called(commitIndex)
}

// Node_SetCommitIndex_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetCommitIndex'
type Node_SetCommitIndex_Call struct {
	*mock.Call
}

// SetCommitIndex is a helper method to define mock.On call
//   - commitIndex uint64
func (_e *Node_Expecter) SetCommitIndex(commitIndex interface{}) *Node_SetCommitIndex_Call {
	return &Node_SetCommitIndex_Call{Call: _e.mock.On("SetCommitIndex", commitIndex)}
}

func (_c *Node_SetCommitIndex_Call) Run(run func(commitIndex uint64)) *Node_SetCommitIndex_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *Node_SetCommitIndex_Call) Return() *Node_SetCommitIndex_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_SetCommitIndex_Call) RunAndReturn(run func(uint64)) *Node_SetCommitIndex_Call {
	_c.Call.Return(run)
	return _c
}

// SetCurrentLeaderId provides a mock function with given fields: leaderId
func (_m *Node) SetCurrentLeaderId(leaderId string) {
	_m.Called(leaderId)
}

// Node_SetCurrentLeaderId_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetCurrentLeaderId'
type Node_SetCurrentLeaderId_Call struct {
	*mock.Call
}

// SetCurrentLeaderId is a helper method to define mock.On call
//   - leaderId string
func (_e *Node_Expecter) SetCurrentLeaderId(leaderId interface{}) *Node_SetCurrentLeaderId_Call {
	return &Node_SetCurrentLeaderId_Call{Call: _e.mock.On("SetCurrentLeaderId", leaderId)}
}

func (_c *Node_SetCurrentLeaderId_Call) Run(run func(leaderId string)) *Node_SetCurrentLeaderId_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Node_SetCurrentLeaderId_Call) Return() *Node_SetCurrentLeaderId_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_SetCurrentLeaderId_Call) RunAndReturn(run func(string)) *Node_SetCurrentLeaderId_Call {
	_c.Call.Return(run)
	return _c
}

// SetCurrentTerm provides a mock function with given fields: newTerm
func (_m *Node) SetCurrentTerm(newTerm uint64) {
	_m.Called(newTerm)
}

// Node_SetCurrentTerm_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetCurrentTerm'
type Node_SetCurrentTerm_Call struct {
	*mock.Call
}

// SetCurrentTerm is a helper method to define mock.On call
//   - newTerm uint64
func (_e *Node_Expecter) SetCurrentTerm(newTerm interface{}) *Node_SetCurrentTerm_Call {
	return &Node_SetCurrentTerm_Call{Call: _e.mock.On("SetCurrentTerm", newTerm)}
}

func (_c *Node_SetCurrentTerm_Call) Run(run func(newTerm uint64)) *Node_SetCurrentTerm_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *Node_SetCurrentTerm_Call) Return() *Node_SetCurrentTerm_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_SetCurrentTerm_Call) RunAndReturn(run func(uint64)) *Node_SetCurrentTerm_Call {
	_c.Call.Return(run)
	return _c
}

// SetState provides a mock function with given fields: newState
func (_m *Node) SetState(newState node.State) {
	_m.Called(newState)
}

// Node_SetState_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetState'
type Node_SetState_Call struct {
	*mock.Call
}

// SetState is a helper method to define mock.On call
//   - newState node.State
func (_e *Node_Expecter) SetState(newState interface{}) *Node_SetState_Call {
	return &Node_SetState_Call{Call: _e.mock.On("SetState", newState)}
}

func (_c *Node_SetState_Call) Run(run func(newState node.State)) *Node_SetState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(node.State))
	})
	return _c
}

func (_c *Node_SetState_Call) Return() *Node_SetState_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_SetState_Call) RunAndReturn(run func(node.State)) *Node_SetState_Call {
	_c.Call.Return(run)
	return _c
}

// SetVotedFor provides a mock function with given fields: peerId
func (_m *Node) SetVotedFor(peerId string) {
	_m.Called(peerId)
}

// Node_SetVotedFor_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetVotedFor'
type Node_SetVotedFor_Call struct {
	*mock.Call
}

// SetVotedFor is a helper method to define mock.On call
//   - peerId string
func (_e *Node_Expecter) SetVotedFor(peerId interface{}) *Node_SetVotedFor_Call {
	return &Node_SetVotedFor_Call{Call: _e.mock.On("SetVotedFor", peerId)}
}

func (_c *Node_SetVotedFor_Call) Run(run func(peerId string)) *Node_SetVotedFor_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Node_SetVotedFor_Call) Return() *Node_SetVotedFor_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_SetVotedFor_Call) RunAndReturn(run func(string)) *Node_SetVotedFor_Call {
	_c.Call.Return(run)
	return _c
}

// SetVotedForTerm provides a mock function with given fields: term, voted
func (_m *Node) SetVotedForTerm(term uint64, voted bool) {
	_m.Called(term, voted)
}

// Node_SetVotedForTerm_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetVotedForTerm'
type Node_SetVotedForTerm_Call struct {
	*mock.Call
}

// SetVotedForTerm is a helper method to define mock.On call
//   - term uint64
//   - voted bool
func (_e *Node_Expecter) SetVotedForTerm(term interface{}, voted interface{}) *Node_SetVotedForTerm_Call {
	return &Node_SetVotedForTerm_Call{Call: _e.mock.On("SetVotedForTerm", term, voted)}
}

func (_c *Node_SetVotedForTerm_Call) Run(run func(term uint64, voted bool)) *Node_SetVotedForTerm_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(bool))
	})
	return _c
}

func (_c *Node_SetVotedForTerm_Call) Return() *Node_SetVotedForTerm_Call {
	_c.Call.Return()
	return _c
}

func (_c *Node_SetVotedForTerm_Call) RunAndReturn(run func(uint64, bool)) *Node_SetVotedForTerm_Call {
	_c.Call.Return(run)
	return _c
}

// VotedForTerm provides a mock function with given fields:
func (_m *Node) VotedForTerm() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for VotedForTerm")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Node_VotedForTerm_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'VotedForTerm'
type Node_VotedForTerm_Call struct {
	*mock.Call
}

// VotedForTerm is a helper method to define mock.On call
func (_e *Node_Expecter) VotedForTerm() *Node_VotedForTerm_Call {
	return &Node_VotedForTerm_Call{Call: _e.mock.On("VotedForTerm")}
}

func (_c *Node_VotedForTerm_Call) Run(run func()) *Node_VotedForTerm_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Node_VotedForTerm_Call) Return(_a0 bool) *Node_VotedForTerm_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Node_VotedForTerm_Call) RunAndReturn(run func() bool) *Node_VotedForTerm_Call {
	_c.Call.Return(run)
	return _c
}

// NewNode creates a new instance of Node. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewNode(t interface {
	mock.TestingT
	Cleanup(func())
}) *Node {
	mock := &Node{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
