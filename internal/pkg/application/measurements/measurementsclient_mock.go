// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package measurements

import (
	"context"
	"sync"
	"time"
)

// Ensure, that MeasurementsClientMock does implement MeasurementsClient.
// If this is not the case, regenerate this file with moq.
var _ MeasurementsClient = &MeasurementsClientMock{}

// MeasurementsClientMock is a mock implementation of MeasurementsClient.
//
//	func TestSomethingThatUsesMeasurementsClient(t *testing.T) {
//
//		// make and configure a mocked MeasurementsClient
//		mockedMeasurementsClient := &MeasurementsClientMock{
//			GetCountTrueValuesFunc: func(ctx context.Context, measurmentID string, timeAt time.Time, endTimeAt time.Time) (float64, error) {
//				panic("mock out the GetCountTrueValues method")
//			},
//			GetMaxValueFunc: func(ctx context.Context, measurmentID string) (float64, error) {
//				panic("mock out the GetMaxValue method")
//			},
//		}
//
//		// use mockedMeasurementsClient in code that requires MeasurementsClient
//		// and then make assertions.
//
//	}
type MeasurementsClientMock struct {
	// GetCountTrueValuesFunc mocks the GetCountTrueValues method.
	GetCountTrueValuesFunc func(ctx context.Context, measurmentID string, timeAt time.Time, endTimeAt time.Time) (float64, error)

	// GetMaxValueFunc mocks the GetMaxValue method.
	GetMaxValueFunc func(ctx context.Context, measurmentID string) (float64, error)

	// calls tracks calls to the methods.
	calls struct {
		// GetCountTrueValues holds details about calls to the GetCountTrueValues method.
		GetCountTrueValues []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// MeasurmentID is the measurmentID argument value.
			MeasurmentID string
			// TimeAt is the timeAt argument value.
			TimeAt time.Time
			// EndTimeAt is the endTimeAt argument value.
			EndTimeAt time.Time
		}
		// GetMaxValue holds details about calls to the GetMaxValue method.
		GetMaxValue []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// MeasurmentID is the measurmentID argument value.
			MeasurmentID string
		}
	}
	lockGetCountTrueValues sync.RWMutex
	lockGetMaxValue        sync.RWMutex
}

// GetCountTrueValues calls GetCountTrueValuesFunc.
func (mock *MeasurementsClientMock) GetCountTrueValues(ctx context.Context, measurmentID string, timeAt time.Time, endTimeAt time.Time) (float64, error) {
	if mock.GetCountTrueValuesFunc == nil {
		panic("MeasurementsClientMock.GetCountTrueValuesFunc: method is nil but MeasurementsClient.GetCountTrueValues was just called")
	}
	callInfo := struct {
		Ctx          context.Context
		MeasurmentID string
		TimeAt       time.Time
		EndTimeAt    time.Time
	}{
		Ctx:          ctx,
		MeasurmentID: measurmentID,
		TimeAt:       timeAt,
		EndTimeAt:    endTimeAt,
	}
	mock.lockGetCountTrueValues.Lock()
	mock.calls.GetCountTrueValues = append(mock.calls.GetCountTrueValues, callInfo)
	mock.lockGetCountTrueValues.Unlock()
	return mock.GetCountTrueValuesFunc(ctx, measurmentID, timeAt, endTimeAt)
}

// GetCountTrueValuesCalls gets all the calls that were made to GetCountTrueValues.
// Check the length with:
//
//	len(mockedMeasurementsClient.GetCountTrueValuesCalls())
func (mock *MeasurementsClientMock) GetCountTrueValuesCalls() []struct {
	Ctx          context.Context
	MeasurmentID string
	TimeAt       time.Time
	EndTimeAt    time.Time
} {
	var calls []struct {
		Ctx          context.Context
		MeasurmentID string
		TimeAt       time.Time
		EndTimeAt    time.Time
	}
	mock.lockGetCountTrueValues.RLock()
	calls = mock.calls.GetCountTrueValues
	mock.lockGetCountTrueValues.RUnlock()
	return calls
}

// GetMaxValue calls GetMaxValueFunc.
func (mock *MeasurementsClientMock) GetMaxValue(ctx context.Context, measurmentID string) (float64, error) {
	if mock.GetMaxValueFunc == nil {
		panic("MeasurementsClientMock.GetMaxValueFunc: method is nil but MeasurementsClient.GetMaxValue was just called")
	}
	callInfo := struct {
		Ctx          context.Context
		MeasurmentID string
	}{
		Ctx:          ctx,
		MeasurmentID: measurmentID,
	}
	mock.lockGetMaxValue.Lock()
	mock.calls.GetMaxValue = append(mock.calls.GetMaxValue, callInfo)
	mock.lockGetMaxValue.Unlock()
	return mock.GetMaxValueFunc(ctx, measurmentID)
}

// GetMaxValueCalls gets all the calls that were made to GetMaxValue.
// Check the length with:
//
//	len(mockedMeasurementsClient.GetMaxValueCalls())
func (mock *MeasurementsClientMock) GetMaxValueCalls() []struct {
	Ctx          context.Context
	MeasurmentID string
} {
	var calls []struct {
		Ctx          context.Context
		MeasurmentID string
	}
	mock.lockGetMaxValue.RLock()
	calls = mock.calls.GetMaxValue
	mock.lockGetMaxValue.RUnlock()
	return calls
}
