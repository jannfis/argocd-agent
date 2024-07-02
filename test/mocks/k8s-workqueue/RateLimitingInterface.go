// Code generated by mockery v2.43.0. DO NOT EDIT.

package mocks

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// RateLimitingInterface is an autogenerated mock type for the RateLimitingInterface type
type RateLimitingInterface struct {
	mock.Mock
}

type RateLimitingInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *RateLimitingInterface) EXPECT() *RateLimitingInterface_Expecter {
	return &RateLimitingInterface_Expecter{mock: &_m.Mock}
}

// Add provides a mock function with given fields: item
func (_m *RateLimitingInterface) Add(item interface{}) {
	_m.Called(item)
}

// RateLimitingInterface_Add_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Add'
type RateLimitingInterface_Add_Call struct {
	*mock.Call
}

// Add is a helper method to define mock.On call
//   - item interface{}
func (_e *RateLimitingInterface_Expecter) Add(item interface{}) *RateLimitingInterface_Add_Call {
	return &RateLimitingInterface_Add_Call{Call: _e.mock.On("Add", item)}
}

func (_c *RateLimitingInterface_Add_Call) Run(run func(item interface{})) *RateLimitingInterface_Add_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *RateLimitingInterface_Add_Call) Return() *RateLimitingInterface_Add_Call {
	_c.Call.Return()
	return _c
}

func (_c *RateLimitingInterface_Add_Call) RunAndReturn(run func(interface{})) *RateLimitingInterface_Add_Call {
	_c.Call.Return(run)
	return _c
}

// AddAfter provides a mock function with given fields: item, duration
func (_m *RateLimitingInterface) AddAfter(item interface{}, duration time.Duration) {
	_m.Called(item, duration)
}

// RateLimitingInterface_AddAfter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddAfter'
type RateLimitingInterface_AddAfter_Call struct {
	*mock.Call
}

// AddAfter is a helper method to define mock.On call
//   - item interface{}
//   - duration time.Duration
func (_e *RateLimitingInterface_Expecter) AddAfter(item interface{}, duration interface{}) *RateLimitingInterface_AddAfter_Call {
	return &RateLimitingInterface_AddAfter_Call{Call: _e.mock.On("AddAfter", item, duration)}
}

func (_c *RateLimitingInterface_AddAfter_Call) Run(run func(item interface{}, duration time.Duration)) *RateLimitingInterface_AddAfter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}), args[1].(time.Duration))
	})
	return _c
}

func (_c *RateLimitingInterface_AddAfter_Call) Return() *RateLimitingInterface_AddAfter_Call {
	_c.Call.Return()
	return _c
}

func (_c *RateLimitingInterface_AddAfter_Call) RunAndReturn(run func(interface{}, time.Duration)) *RateLimitingInterface_AddAfter_Call {
	_c.Call.Return(run)
	return _c
}

// AddRateLimited provides a mock function with given fields: item
func (_m *RateLimitingInterface) AddRateLimited(item interface{}) {
	_m.Called(item)
}

// RateLimitingInterface_AddRateLimited_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddRateLimited'
type RateLimitingInterface_AddRateLimited_Call struct {
	*mock.Call
}

// AddRateLimited is a helper method to define mock.On call
//   - item interface{}
func (_e *RateLimitingInterface_Expecter) AddRateLimited(item interface{}) *RateLimitingInterface_AddRateLimited_Call {
	return &RateLimitingInterface_AddRateLimited_Call{Call: _e.mock.On("AddRateLimited", item)}
}

func (_c *RateLimitingInterface_AddRateLimited_Call) Run(run func(item interface{})) *RateLimitingInterface_AddRateLimited_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *RateLimitingInterface_AddRateLimited_Call) Return() *RateLimitingInterface_AddRateLimited_Call {
	_c.Call.Return()
	return _c
}

func (_c *RateLimitingInterface_AddRateLimited_Call) RunAndReturn(run func(interface{})) *RateLimitingInterface_AddRateLimited_Call {
	_c.Call.Return(run)
	return _c
}

// Done provides a mock function with given fields: item
func (_m *RateLimitingInterface) Done(item interface{}) {
	_m.Called(item)
}

// RateLimitingInterface_Done_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Done'
type RateLimitingInterface_Done_Call struct {
	*mock.Call
}

// Done is a helper method to define mock.On call
//   - item interface{}
func (_e *RateLimitingInterface_Expecter) Done(item interface{}) *RateLimitingInterface_Done_Call {
	return &RateLimitingInterface_Done_Call{Call: _e.mock.On("Done", item)}
}

func (_c *RateLimitingInterface_Done_Call) Run(run func(item interface{})) *RateLimitingInterface_Done_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *RateLimitingInterface_Done_Call) Return() *RateLimitingInterface_Done_Call {
	_c.Call.Return()
	return _c
}

func (_c *RateLimitingInterface_Done_Call) RunAndReturn(run func(interface{})) *RateLimitingInterface_Done_Call {
	_c.Call.Return(run)
	return _c
}

// Forget provides a mock function with given fields: item
func (_m *RateLimitingInterface) Forget(item interface{}) {
	_m.Called(item)
}

// RateLimitingInterface_Forget_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Forget'
type RateLimitingInterface_Forget_Call struct {
	*mock.Call
}

// Forget is a helper method to define mock.On call
//   - item interface{}
func (_e *RateLimitingInterface_Expecter) Forget(item interface{}) *RateLimitingInterface_Forget_Call {
	return &RateLimitingInterface_Forget_Call{Call: _e.mock.On("Forget", item)}
}

func (_c *RateLimitingInterface_Forget_Call) Run(run func(item interface{})) *RateLimitingInterface_Forget_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *RateLimitingInterface_Forget_Call) Return() *RateLimitingInterface_Forget_Call {
	_c.Call.Return()
	return _c
}

func (_c *RateLimitingInterface_Forget_Call) RunAndReturn(run func(interface{})) *RateLimitingInterface_Forget_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields:
func (_m *RateLimitingInterface) Get() (interface{}, bool) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 interface{}
	var r1 bool
	if rf, ok := ret.Get(0).(func() (interface{}, bool)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	if rf, ok := ret.Get(1).(func() bool); ok {
		r1 = rf()
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// RateLimitingInterface_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type RateLimitingInterface_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
func (_e *RateLimitingInterface_Expecter) Get() *RateLimitingInterface_Get_Call {
	return &RateLimitingInterface_Get_Call{Call: _e.mock.On("Get")}
}

func (_c *RateLimitingInterface_Get_Call) Run(run func()) *RateLimitingInterface_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *RateLimitingInterface_Get_Call) Return(item interface{}, shutdown bool) *RateLimitingInterface_Get_Call {
	_c.Call.Return(item, shutdown)
	return _c
}

func (_c *RateLimitingInterface_Get_Call) RunAndReturn(run func() (interface{}, bool)) *RateLimitingInterface_Get_Call {
	_c.Call.Return(run)
	return _c
}

// Len provides a mock function with given fields:
func (_m *RateLimitingInterface) Len() int {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Len")
	}

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// RateLimitingInterface_Len_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Len'
type RateLimitingInterface_Len_Call struct {
	*mock.Call
}

// Len is a helper method to define mock.On call
func (_e *RateLimitingInterface_Expecter) Len() *RateLimitingInterface_Len_Call {
	return &RateLimitingInterface_Len_Call{Call: _e.mock.On("Len")}
}

func (_c *RateLimitingInterface_Len_Call) Run(run func()) *RateLimitingInterface_Len_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *RateLimitingInterface_Len_Call) Return(_a0 int) *RateLimitingInterface_Len_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *RateLimitingInterface_Len_Call) RunAndReturn(run func() int) *RateLimitingInterface_Len_Call {
	_c.Call.Return(run)
	return _c
}

// NumRequeues provides a mock function with given fields: item
func (_m *RateLimitingInterface) NumRequeues(item interface{}) int {
	ret := _m.Called(item)

	if len(ret) == 0 {
		panic("no return value specified for NumRequeues")
	}

	var r0 int
	if rf, ok := ret.Get(0).(func(interface{}) int); ok {
		r0 = rf(item)
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// RateLimitingInterface_NumRequeues_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'NumRequeues'
type RateLimitingInterface_NumRequeues_Call struct {
	*mock.Call
}

// NumRequeues is a helper method to define mock.On call
//   - item interface{}
func (_e *RateLimitingInterface_Expecter) NumRequeues(item interface{}) *RateLimitingInterface_NumRequeues_Call {
	return &RateLimitingInterface_NumRequeues_Call{Call: _e.mock.On("NumRequeues", item)}
}

func (_c *RateLimitingInterface_NumRequeues_Call) Run(run func(item interface{})) *RateLimitingInterface_NumRequeues_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *RateLimitingInterface_NumRequeues_Call) Return(_a0 int) *RateLimitingInterface_NumRequeues_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *RateLimitingInterface_NumRequeues_Call) RunAndReturn(run func(interface{}) int) *RateLimitingInterface_NumRequeues_Call {
	_c.Call.Return(run)
	return _c
}

// ShutDown provides a mock function with given fields:
func (_m *RateLimitingInterface) ShutDown() {
	_m.Called()
}

// RateLimitingInterface_ShutDown_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ShutDown'
type RateLimitingInterface_ShutDown_Call struct {
	*mock.Call
}

// ShutDown is a helper method to define mock.On call
func (_e *RateLimitingInterface_Expecter) ShutDown() *RateLimitingInterface_ShutDown_Call {
	return &RateLimitingInterface_ShutDown_Call{Call: _e.mock.On("ShutDown")}
}

func (_c *RateLimitingInterface_ShutDown_Call) Run(run func()) *RateLimitingInterface_ShutDown_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *RateLimitingInterface_ShutDown_Call) Return() *RateLimitingInterface_ShutDown_Call {
	_c.Call.Return()
	return _c
}

func (_c *RateLimitingInterface_ShutDown_Call) RunAndReturn(run func()) *RateLimitingInterface_ShutDown_Call {
	_c.Call.Return(run)
	return _c
}

// ShutDownWithDrain provides a mock function with given fields:
func (_m *RateLimitingInterface) ShutDownWithDrain() {
	_m.Called()
}

// RateLimitingInterface_ShutDownWithDrain_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ShutDownWithDrain'
type RateLimitingInterface_ShutDownWithDrain_Call struct {
	*mock.Call
}

// ShutDownWithDrain is a helper method to define mock.On call
func (_e *RateLimitingInterface_Expecter) ShutDownWithDrain() *RateLimitingInterface_ShutDownWithDrain_Call {
	return &RateLimitingInterface_ShutDownWithDrain_Call{Call: _e.mock.On("ShutDownWithDrain")}
}

func (_c *RateLimitingInterface_ShutDownWithDrain_Call) Run(run func()) *RateLimitingInterface_ShutDownWithDrain_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *RateLimitingInterface_ShutDownWithDrain_Call) Return() *RateLimitingInterface_ShutDownWithDrain_Call {
	_c.Call.Return()
	return _c
}

func (_c *RateLimitingInterface_ShutDownWithDrain_Call) RunAndReturn(run func()) *RateLimitingInterface_ShutDownWithDrain_Call {
	_c.Call.Return(run)
	return _c
}

// ShuttingDown provides a mock function with given fields:
func (_m *RateLimitingInterface) ShuttingDown() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ShuttingDown")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// RateLimitingInterface_ShuttingDown_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ShuttingDown'
type RateLimitingInterface_ShuttingDown_Call struct {
	*mock.Call
}

// ShuttingDown is a helper method to define mock.On call
func (_e *RateLimitingInterface_Expecter) ShuttingDown() *RateLimitingInterface_ShuttingDown_Call {
	return &RateLimitingInterface_ShuttingDown_Call{Call: _e.mock.On("ShuttingDown")}
}

func (_c *RateLimitingInterface_ShuttingDown_Call) Run(run func()) *RateLimitingInterface_ShuttingDown_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *RateLimitingInterface_ShuttingDown_Call) Return(_a0 bool) *RateLimitingInterface_ShuttingDown_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *RateLimitingInterface_ShuttingDown_Call) RunAndReturn(run func() bool) *RateLimitingInterface_ShuttingDown_Call {
	_c.Call.Return(run)
	return _c
}

// NewRateLimitingInterface creates a new instance of RateLimitingInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRateLimitingInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *RateLimitingInterface {
	mock := &RateLimitingInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
