// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package database

import (
	"context"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
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
//			AddSettingFunc: func(ctx context.Context, id string, s functions.Setting) error {
//				panic("mock out the AddSetting method")
//			},
//			GetSettingsFunc: func(ctx context.Context) ([]functions.Setting, error) {
//				panic("mock out the GetSettings method")
//			},
//			InitializeFunc: func(contextMoqParam context.Context) error {
//				panic("mock out the Initialize method")
//			},
//			LoadStateFunc: func(ctx context.Context, id string) ([]byte, error) {
//				panic("mock out the LoadState method")
//			},
//			SaveStateFunc: func(ctx context.Context, id string, a any) error {
//				panic("mock out the SaveState method")
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

	// AddSettingFunc mocks the AddSetting method.
	AddSettingFunc func(ctx context.Context, id string, s functions.Setting) error

	// GetSettingsFunc mocks the GetSettings method.
	GetSettingsFunc func(ctx context.Context) ([]functions.Setting, error)

	// InitializeFunc mocks the Initialize method.
	InitializeFunc func(contextMoqParam context.Context) error

	// LoadStateFunc mocks the LoadState method.
	LoadStateFunc func(ctx context.Context, id string) ([]byte, error)

	// SaveStateFunc mocks the SaveState method.
	SaveStateFunc func(ctx context.Context, id string, a any) error

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
		// AddSetting holds details about calls to the AddSetting method.
		AddSetting []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// S is the s argument value.
			S functions.Setting
		}
		// GetSettings holds details about calls to the GetSettings method.
		GetSettings []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// Initialize holds details about calls to the Initialize method.
		Initialize []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
		}
		// LoadState holds details about calls to the LoadState method.
		LoadState []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// SaveState holds details about calls to the SaveState method.
		SaveState []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// A is the a argument value.
			A any
		}
	}
	lockAdd         sync.RWMutex
	lockAddSetting  sync.RWMutex
	lockGetSettings sync.RWMutex
	lockInitialize  sync.RWMutex
	lockLoadState   sync.RWMutex
	lockSaveState   sync.RWMutex
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

// AddSetting calls AddSettingFunc.
func (mock *StorageMock) AddSetting(ctx context.Context, id string, s functions.Setting) error {
	if mock.AddSettingFunc == nil {
		panic("StorageMock.AddSettingFunc: method is nil but Storage.AddSetting was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
		S   functions.Setting
	}{
		Ctx: ctx,
		ID:  id,
		S:   s,
	}
	mock.lockAddSetting.Lock()
	mock.calls.AddSetting = append(mock.calls.AddSetting, callInfo)
	mock.lockAddSetting.Unlock()
	return mock.AddSettingFunc(ctx, id, s)
}

// AddSettingCalls gets all the calls that were made to AddSetting.
// Check the length with:
//
//	len(mockedStorage.AddSettingCalls())
func (mock *StorageMock) AddSettingCalls() []struct {
	Ctx context.Context
	ID  string
	S   functions.Setting
} {
	var calls []struct {
		Ctx context.Context
		ID  string
		S   functions.Setting
	}
	mock.lockAddSetting.RLock()
	calls = mock.calls.AddSetting
	mock.lockAddSetting.RUnlock()
	return calls
}

// GetSettings calls GetSettingsFunc.
func (mock *StorageMock) GetSettings(ctx context.Context) ([]functions.Setting, error) {
	if mock.GetSettingsFunc == nil {
		panic("StorageMock.GetSettingsFunc: method is nil but Storage.GetSettings was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockGetSettings.Lock()
	mock.calls.GetSettings = append(mock.calls.GetSettings, callInfo)
	mock.lockGetSettings.Unlock()
	return mock.GetSettingsFunc(ctx)
}

// GetSettingsCalls gets all the calls that were made to GetSettings.
// Check the length with:
//
//	len(mockedStorage.GetSettingsCalls())
func (mock *StorageMock) GetSettingsCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockGetSettings.RLock()
	calls = mock.calls.GetSettings
	mock.lockGetSettings.RUnlock()
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

// LoadState calls LoadStateFunc.
func (mock *StorageMock) LoadState(ctx context.Context, id string) ([]byte, error) {
	if mock.LoadStateFunc == nil {
		panic("StorageMock.LoadStateFunc: method is nil but Storage.LoadState was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	mock.lockLoadState.Lock()
	mock.calls.LoadState = append(mock.calls.LoadState, callInfo)
	mock.lockLoadState.Unlock()
	return mock.LoadStateFunc(ctx, id)
}

// LoadStateCalls gets all the calls that were made to LoadState.
// Check the length with:
//
//	len(mockedStorage.LoadStateCalls())
func (mock *StorageMock) LoadStateCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	mock.lockLoadState.RLock()
	calls = mock.calls.LoadState
	mock.lockLoadState.RUnlock()
	return calls
}

// SaveState calls SaveStateFunc.
func (mock *StorageMock) SaveState(ctx context.Context, id string, a any) error {
	if mock.SaveStateFunc == nil {
		panic("StorageMock.SaveStateFunc: method is nil but Storage.SaveState was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
		A   any
	}{
		Ctx: ctx,
		ID:  id,
		A:   a,
	}
	mock.lockSaveState.Lock()
	mock.calls.SaveState = append(mock.calls.SaveState, callInfo)
	mock.lockSaveState.Unlock()
	return mock.SaveStateFunc(ctx, id, a)
}

// SaveStateCalls gets all the calls that were made to SaveState.
// Check the length with:
//
//	len(mockedStorage.SaveStateCalls())
func (mock *StorageMock) SaveStateCalls() []struct {
	Ctx context.Context
	ID  string
	A   any
} {
	var calls []struct {
		Ctx context.Context
		ID  string
		A   any
	}
	mock.lockSaveState.RLock()
	calls = mock.calls.SaveState
	mock.lockSaveState.RUnlock()
	return calls
}
