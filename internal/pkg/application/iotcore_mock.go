// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package application

import (
	"context"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"sync"
)

// Ensure, that AppMock does implement App.
// If this is not the case, regenerate this file with moq.
var _ App = &AppMock{}

// AppMock is a mock implementation of App.
//
//	func TestSomethingThatUsesApp(t *testing.T) {
//
//		// make and configure a mocked App
//		mockedApp := &AppMock{
//			MessageAcceptedFunc: func(ctx context.Context, evt events.MessageAccepted) error {
//				panic("mock out the MessageAccepted method")
//			},
//			MessageReceivedFunc: func(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
//				panic("mock out the MessageReceived method")
//			},
//			QueryFunc: func(ctx context.Context, params map[string]any) ([]functions.Function, error) {
//				panic("mock out the Query method")
//			},
//			RegisterFunc: func(ctx context.Context, s functions.Setting) error {
//				panic("mock out the Register method")
//			},
//		}
//
//		// use mockedApp in code that requires App
//		// and then make assertions.
//
//	}
type AppMock struct {
	// MessageAcceptedFunc mocks the MessageAccepted method.
	MessageAcceptedFunc func(ctx context.Context, evt events.MessageAccepted) error

	// MessageReceivedFunc mocks the MessageReceived method.
	MessageReceivedFunc func(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)

	// QueryFunc mocks the Query method.
	QueryFunc func(ctx context.Context, params map[string]any) ([]functions.Function, error)

	// RegisterFunc mocks the Register method.
	RegisterFunc func(ctx context.Context, s functions.Setting) error

	// calls tracks calls to the methods.
	calls struct {
		// MessageAccepted holds details about calls to the MessageAccepted method.
		MessageAccepted []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Evt is the evt argument value.
			Evt events.MessageAccepted
		}
		// MessageReceived holds details about calls to the MessageReceived method.
		MessageReceived []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Msg is the msg argument value.
			Msg events.MessageReceived
		}
		// Query holds details about calls to the Query method.
		Query []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Params is the params argument value.
			Params map[string]any
		}
		// Register holds details about calls to the Register method.
		Register []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// S is the s argument value.
			S functions.Setting
		}
	}
	lockMessageAccepted sync.RWMutex
	lockMessageReceived sync.RWMutex
	lockQuery           sync.RWMutex
	lockRegister        sync.RWMutex
}

// MessageAccepted calls MessageAcceptedFunc.
func (mock *AppMock) MessageAccepted(ctx context.Context, evt events.MessageAccepted) error {
	if mock.MessageAcceptedFunc == nil {
		panic("AppMock.MessageAcceptedFunc: method is nil but App.MessageAccepted was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Evt events.MessageAccepted
	}{
		Ctx: ctx,
		Evt: evt,
	}
	mock.lockMessageAccepted.Lock()
	mock.calls.MessageAccepted = append(mock.calls.MessageAccepted, callInfo)
	mock.lockMessageAccepted.Unlock()
	return mock.MessageAcceptedFunc(ctx, evt)
}

// MessageAcceptedCalls gets all the calls that were made to MessageAccepted.
// Check the length with:
//
//	len(mockedApp.MessageAcceptedCalls())
func (mock *AppMock) MessageAcceptedCalls() []struct {
	Ctx context.Context
	Evt events.MessageAccepted
} {
	var calls []struct {
		Ctx context.Context
		Evt events.MessageAccepted
	}
	mock.lockMessageAccepted.RLock()
	calls = mock.calls.MessageAccepted
	mock.lockMessageAccepted.RUnlock()
	return calls
}

// MessageReceived calls MessageReceivedFunc.
func (mock *AppMock) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	if mock.MessageReceivedFunc == nil {
		panic("AppMock.MessageReceivedFunc: method is nil but App.MessageReceived was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Msg events.MessageReceived
	}{
		Ctx: ctx,
		Msg: msg,
	}
	mock.lockMessageReceived.Lock()
	mock.calls.MessageReceived = append(mock.calls.MessageReceived, callInfo)
	mock.lockMessageReceived.Unlock()
	return mock.MessageReceivedFunc(ctx, msg)
}

// MessageReceivedCalls gets all the calls that were made to MessageReceived.
// Check the length with:
//
//	len(mockedApp.MessageReceivedCalls())
func (mock *AppMock) MessageReceivedCalls() []struct {
	Ctx context.Context
	Msg events.MessageReceived
} {
	var calls []struct {
		Ctx context.Context
		Msg events.MessageReceived
	}
	mock.lockMessageReceived.RLock()
	calls = mock.calls.MessageReceived
	mock.lockMessageReceived.RUnlock()
	return calls
}

// Query calls QueryFunc.
func (mock *AppMock) Query(ctx context.Context, params map[string]any) ([]functions.Function, error) {
	if mock.QueryFunc == nil {
		panic("AppMock.QueryFunc: method is nil but App.Query was just called")
	}
	callInfo := struct {
		Ctx    context.Context
		Params map[string]any
	}{
		Ctx:    ctx,
		Params: params,
	}
	mock.lockQuery.Lock()
	mock.calls.Query = append(mock.calls.Query, callInfo)
	mock.lockQuery.Unlock()
	return mock.QueryFunc(ctx, params)
}

// QueryCalls gets all the calls that were made to Query.
// Check the length with:
//
//	len(mockedApp.QueryCalls())
func (mock *AppMock) QueryCalls() []struct {
	Ctx    context.Context
	Params map[string]any
} {
	var calls []struct {
		Ctx    context.Context
		Params map[string]any
	}
	mock.lockQuery.RLock()
	calls = mock.calls.Query
	mock.lockQuery.RUnlock()
	return calls
}

// Register calls RegisterFunc.
func (mock *AppMock) Register(ctx context.Context, s functions.Setting) error {
	if mock.RegisterFunc == nil {
		panic("AppMock.RegisterFunc: method is nil but App.Register was just called")
	}
	callInfo := struct {
		Ctx context.Context
		S   functions.Setting
	}{
		Ctx: ctx,
		S:   s,
	}
	mock.lockRegister.Lock()
	mock.calls.Register = append(mock.calls.Register, callInfo)
	mock.lockRegister.Unlock()
	return mock.RegisterFunc(ctx, s)
}

// RegisterCalls gets all the calls that were made to Register.
// Check the length with:
//
//	len(mockedApp.RegisterCalls())
func (mock *AppMock) RegisterCalls() []struct {
	Ctx context.Context
	S   functions.Setting
} {
	var calls []struct {
		Ctx context.Context
		S   functions.Setting
	}
	mock.lockRegister.RLock()
	calls = mock.calls.Register
	mock.lockRegister.RUnlock()
	return calls
}
