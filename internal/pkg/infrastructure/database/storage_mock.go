// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package database

import (
	"context"
	"sync"
	"time"
)

// Ensure, that StorageMock does implement Storage.
// If this is not the case, regenerate this file with moq.
var _ Storage = &StorageMock{}

// StorageMock is a mock implementation of Storage.
//
//	func TestSomethingThatUsesStorage(t *testing.T) {
//
//		// make and configure a mocked Storage
//		mockedStorage := &StorageMock{
//			AddFunc: func(ctx context.Context, id string, label string, value float64, timestamp time.Time) error {
//				panic("mock out the Add method")
//			},
//			AddFnFunc: func(ctx context.Context, id string, fnType string, subType string, tenant string, source string, lat float64, lon float64) error {
//				panic("mock out the AddFn method")
//			},
//			HistoryFunc: func(ctx context.Context, id string, label string, lastN int) ([]LogValue, error) {
//				panic("mock out the History method")
//			},
//			InitializeFunc: func(contextMoqParam context.Context) error {
//				panic("mock out the Initialize method")
//			},
//		}
//
//		// use mockedStorage in code that requires Storage
//		// and then make assertions.
//
//	}
type StorageMock struct {
	// AddFunc mocks the Add method.
	AddFunc func(ctx context.Context, id string, label string, value float64, timestamp time.Time) error

	// AddFnFunc mocks the AddFn method.
	AddFnFunc func(ctx context.Context, id string, fnType string, subType string, tenant string, source string, lat float64, lon float64) error

	// HistoryFunc mocks the History method.
	HistoryFunc func(ctx context.Context, id string, label string, lastN int) ([]LogValue, error)

	// InitializeFunc mocks the Initialize method.
	InitializeFunc func(contextMoqParam context.Context) error

	// calls tracks calls to the methods.
	calls struct {
		// Add holds details about calls to the Add method.
		Add []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Label is the label argument value.
			Label string
			// Value is the value argument value.
			Value float64
			// Timestamp is the timestamp argument value.
			Timestamp time.Time
		}
		// AddFn holds details about calls to the AddFn method.
		AddFn []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// FnType is the fnType argument value.
			FnType string
			// SubType is the subType argument value.
			SubType string
			// Tenant is the tenant argument value.
			Tenant string
			// Source is the source argument value.
			Source string
			// Lat is the lat argument value.
			Lat float64
			// Lon is the lon argument value.
			Lon float64
		}
		// History holds details about calls to the History method.
		History []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Label is the label argument value.
			Label string
			// LastN is the lastN argument value.
			LastN int
		}
		// Initialize holds details about calls to the Initialize method.
		Initialize []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
		}
	}
	lockAdd        sync.RWMutex
	lockAddFn      sync.RWMutex
	lockHistory    sync.RWMutex
	lockInitialize sync.RWMutex
}

// Add calls AddFunc.
func (mock *StorageMock) Add(ctx context.Context, id string, label string, value float64, timestamp time.Time) error {
	if mock.AddFunc == nil {
		panic("StorageMock.AddFunc: method is nil but Storage.Add was just called")
	}
	callInfo := struct {
		Ctx       context.Context
		ID        string
		Label     string
		Value     float64
		Timestamp time.Time
	}{
		Ctx:       ctx,
		ID:        id,
		Label:     label,
		Value:     value,
		Timestamp: timestamp,
	}
	mock.lockAdd.Lock()
	mock.calls.Add = append(mock.calls.Add, callInfo)
	mock.lockAdd.Unlock()
	return mock.AddFunc(ctx, id, label, value, timestamp)
}

// AddCalls gets all the calls that were made to Add.
// Check the length with:
//
//	len(mockedStorage.AddCalls())
func (mock *StorageMock) AddCalls() []struct {
	Ctx       context.Context
	ID        string
	Label     string
	Value     float64
	Timestamp time.Time
} {
	var calls []struct {
		Ctx       context.Context
		ID        string
		Label     string
		Value     float64
		Timestamp time.Time
	}
	mock.lockAdd.RLock()
	calls = mock.calls.Add
	mock.lockAdd.RUnlock()
	return calls
}

// AddFnct calls AddFnFunc.
func (mock *StorageMock) AddFnct(ctx context.Context, id string, fnType string, subType string, tenant string, source string, lat float64, lon float64) error {
	if mock.AddFnFunc == nil {
		panic("StorageMock.AddFnFunc: method is nil but Storage.AddFn was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		ID      string
		FnType  string
		SubType string
		Tenant  string
		Source  string
		Lat     float64
		Lon     float64
	}{
		Ctx:     ctx,
		ID:      id,
		FnType:  fnType,
		SubType: subType,
		Tenant:  tenant,
		Source:  source,
		Lat:     lat,
		Lon:     lon,
	}
	mock.lockAddFn.Lock()
	mock.calls.AddFn = append(mock.calls.AddFn, callInfo)
	mock.lockAddFn.Unlock()
	return mock.AddFnFunc(ctx, id, fnType, subType, tenant, source, lat, lon)
}

// AddFnCalls gets all the calls that were made to AddFn.
// Check the length with:
//
//	len(mockedStorage.AddFnCalls())
func (mock *StorageMock) AddFnCalls() []struct {
	Ctx     context.Context
	ID      string
	FnType  string
	SubType string
	Tenant  string
	Source  string
	Lat     float64
	Lon     float64
} {
	var calls []struct {
		Ctx     context.Context
		ID      string
		FnType  string
		SubType string
		Tenant  string
		Source  string
		Lat     float64
		Lon     float64
	}
	mock.lockAddFn.RLock()
	calls = mock.calls.AddFn
	mock.lockAddFn.RUnlock()
	return calls
}

// History calls HistoryFunc.
func (mock *StorageMock) History(ctx context.Context, id string, label string, lastN int) ([]LogValue, error) {
	if mock.HistoryFunc == nil {
		panic("StorageMock.HistoryFunc: method is nil but Storage.History was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		ID    string
		Label string
		LastN int
	}{
		Ctx:   ctx,
		ID:    id,
		Label: label,
		LastN: lastN,
	}
	mock.lockHistory.Lock()
	mock.calls.History = append(mock.calls.History, callInfo)
	mock.lockHistory.Unlock()
	return mock.HistoryFunc(ctx, id, label, lastN)
}

// HistoryCalls gets all the calls that were made to History.
// Check the length with:
//
//	len(mockedStorage.HistoryCalls())
func (mock *StorageMock) HistoryCalls() []struct {
	Ctx   context.Context
	ID    string
	Label string
	LastN int
} {
	var calls []struct {
		Ctx   context.Context
		ID    string
		Label string
		LastN int
	}
	mock.lockHistory.RLock()
	calls = mock.calls.History
	mock.lockHistory.RUnlock()
	return calls
}

// Initialize calls InitializeFunc.
func (mock *StorageMock) Initialize(contextMoqParam context.Context) error {
	if mock.InitializeFunc == nil {
		panic("StorageMock.InitializeFunc: method is nil but Storage.Initialize was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
	}{
		ContextMoqParam: contextMoqParam,
	}
	mock.lockInitialize.Lock()
	mock.calls.Initialize = append(mock.calls.Initialize, callInfo)
	mock.lockInitialize.Unlock()
	return mock.InitializeFunc(contextMoqParam)
}

// InitializeCalls gets all the calls that were made to Initialize.
// Check the length with:
//
//	len(mockedStorage.InitializeCalls())
func (mock *StorageMock) InitializeCalls() []struct {
	ContextMoqParam context.Context
} {
	var calls []struct {
		ContextMoqParam context.Context
	}
	mock.lockInitialize.RLock()
	calls = mock.calls.Initialize
	mock.lockInitialize.RUnlock()
	return calls
}
