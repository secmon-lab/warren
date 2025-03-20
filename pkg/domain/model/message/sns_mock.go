// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package message

import (
	"net/http"
	"sync"
)

// Ensure, that HTTPClientMock does implement HTTPClient.
// If this is not the case, regenerate this file with moq.
var _ HTTPClient = &HTTPClientMock{}

// HTTPClientMock is a mock implementation of HTTPClient.
//
//	func TestSomethingThatUsesHTTPClient(t *testing.T) {
//
//		// make and configure a mocked HTTPClient
//		mockedHTTPClient := &HTTPClientMock{
//			GetFunc: func(url string) (*http.Response, error) {
//				panic("mock out the Get method")
//			},
//		}
//
//		// use mockedHTTPClient in code that requires HTTPClient
//		// and then make assertions.
//
//	}
type HTTPClientMock struct {
	// GetFunc mocks the Get method.
	GetFunc func(url string) (*http.Response, error)

	// calls tracks calls to the methods.
	calls struct {
		// Get holds details about calls to the Get method.
		Get []struct {
			// URL is the url argument value.
			URL string
		}
	}
	lockGet sync.RWMutex
}

// Get calls GetFunc.
func (mock *HTTPClientMock) Get(url string) (*http.Response, error) {
	if mock.GetFunc == nil {
		panic("HTTPClientMock.GetFunc: method is nil but HTTPClient.Get was just called")
	}
	callInfo := struct {
		URL string
	}{
		URL: url,
	}
	mock.lockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	mock.lockGet.Unlock()
	return mock.GetFunc(url)
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//
//	len(mockedHTTPClient.GetCalls())
func (mock *HTTPClientMock) GetCalls() []struct {
	URL string
} {
	var calls []struct {
		URL string
	}
	mock.lockGet.RLock()
	calls = mock.calls.Get
	mock.lockGet.RUnlock()
	return calls
}
