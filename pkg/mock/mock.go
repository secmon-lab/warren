// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"cloud.google.com/go/vertexai/genai"
	"context"
	"github.com/m-mizutani/opac"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"net/http"
	"sync"
	"time"
)

// Ensure, that SlackServiceMock does implement interfaces.SlackService.
// If this is not the case, regenerate this file with moq.
var _ interfaces.SlackService = &SlackServiceMock{}

// SlackServiceMock is a mock implementation of interfaces.SlackService.
//
//	func TestSomethingThatUsesSlackService(t *testing.T) {
//
//		// make and configure a mocked interfaces.SlackService
//		mockedSlackService := &SlackServiceMock{
//			PostAlertFunc: func(ctx context.Context, alert model.Alert) (string, string, error) {
//				panic("mock out the PostAlert method")
//			},
//			UpdateAlertFunc: func(ctx context.Context, alert model.Alert) error {
//				panic("mock out the UpdateAlert method")
//			},
//			VerifyRequestFunc: func(header http.Header, body []byte) error {
//				panic("mock out the VerifyRequest method")
//			},
//		}
//
//		// use mockedSlackService in code that requires interfaces.SlackService
//		// and then make assertions.
//
//	}
type SlackServiceMock struct {
	// PostAlertFunc mocks the PostAlert method.
	PostAlertFunc func(ctx context.Context, alert model.Alert) (string, string, error)

	// UpdateAlertFunc mocks the UpdateAlert method.
	UpdateAlertFunc func(ctx context.Context, alert model.Alert) error

	// VerifyRequestFunc mocks the VerifyRequest method.
	VerifyRequestFunc func(header http.Header, body []byte) error

	// calls tracks calls to the methods.
	calls struct {
		// PostAlert holds details about calls to the PostAlert method.
		PostAlert []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Alert is the alert argument value.
			Alert model.Alert
		}
		// UpdateAlert holds details about calls to the UpdateAlert method.
		UpdateAlert []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Alert is the alert argument value.
			Alert model.Alert
		}
		// VerifyRequest holds details about calls to the VerifyRequest method.
		VerifyRequest []struct {
			// Header is the header argument value.
			Header http.Header
			// Body is the body argument value.
			Body []byte
		}
	}
	lockPostAlert     sync.RWMutex
	lockUpdateAlert   sync.RWMutex
	lockVerifyRequest sync.RWMutex
}

// PostAlert calls PostAlertFunc.
func (mock *SlackServiceMock) PostAlert(ctx context.Context, alert model.Alert) (string, string, error) {
	if mock.PostAlertFunc == nil {
		panic("SlackServiceMock.PostAlertFunc: method is nil but SlackService.PostAlert was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Alert model.Alert
	}{
		Ctx:   ctx,
		Alert: alert,
	}
	mock.lockPostAlert.Lock()
	mock.calls.PostAlert = append(mock.calls.PostAlert, callInfo)
	mock.lockPostAlert.Unlock()
	return mock.PostAlertFunc(ctx, alert)
}

// PostAlertCalls gets all the calls that were made to PostAlert.
// Check the length with:
//
//	len(mockedSlackService.PostAlertCalls())
func (mock *SlackServiceMock) PostAlertCalls() []struct {
	Ctx   context.Context
	Alert model.Alert
} {
	var calls []struct {
		Ctx   context.Context
		Alert model.Alert
	}
	mock.lockPostAlert.RLock()
	calls = mock.calls.PostAlert
	mock.lockPostAlert.RUnlock()
	return calls
}

// UpdateAlert calls UpdateAlertFunc.
func (mock *SlackServiceMock) UpdateAlert(ctx context.Context, alert model.Alert) error {
	if mock.UpdateAlertFunc == nil {
		panic("SlackServiceMock.UpdateAlertFunc: method is nil but SlackService.UpdateAlert was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Alert model.Alert
	}{
		Ctx:   ctx,
		Alert: alert,
	}
	mock.lockUpdateAlert.Lock()
	mock.calls.UpdateAlert = append(mock.calls.UpdateAlert, callInfo)
	mock.lockUpdateAlert.Unlock()
	return mock.UpdateAlertFunc(ctx, alert)
}

// UpdateAlertCalls gets all the calls that were made to UpdateAlert.
// Check the length with:
//
//	len(mockedSlackService.UpdateAlertCalls())
func (mock *SlackServiceMock) UpdateAlertCalls() []struct {
	Ctx   context.Context
	Alert model.Alert
} {
	var calls []struct {
		Ctx   context.Context
		Alert model.Alert
	}
	mock.lockUpdateAlert.RLock()
	calls = mock.calls.UpdateAlert
	mock.lockUpdateAlert.RUnlock()
	return calls
}

// VerifyRequest calls VerifyRequestFunc.
func (mock *SlackServiceMock) VerifyRequest(header http.Header, body []byte) error {
	if mock.VerifyRequestFunc == nil {
		panic("SlackServiceMock.VerifyRequestFunc: method is nil but SlackService.VerifyRequest was just called")
	}
	callInfo := struct {
		Header http.Header
		Body   []byte
	}{
		Header: header,
		Body:   body,
	}
	mock.lockVerifyRequest.Lock()
	mock.calls.VerifyRequest = append(mock.calls.VerifyRequest, callInfo)
	mock.lockVerifyRequest.Unlock()
	return mock.VerifyRequestFunc(header, body)
}

// VerifyRequestCalls gets all the calls that were made to VerifyRequest.
// Check the length with:
//
//	len(mockedSlackService.VerifyRequestCalls())
func (mock *SlackServiceMock) VerifyRequestCalls() []struct {
	Header http.Header
	Body   []byte
} {
	var calls []struct {
		Header http.Header
		Body   []byte
	}
	mock.lockVerifyRequest.RLock()
	calls = mock.calls.VerifyRequest
	mock.lockVerifyRequest.RUnlock()
	return calls
}

// Ensure, that GenAIChatSessionMock does implement interfaces.GenAIChatSession.
// If this is not the case, regenerate this file with moq.
var _ interfaces.GenAIChatSession = &GenAIChatSessionMock{}

// GenAIChatSessionMock is a mock implementation of interfaces.GenAIChatSession.
//
//	func TestSomethingThatUsesGenAIChatSession(t *testing.T) {
//
//		// make and configure a mocked interfaces.GenAIChatSession
//		mockedGenAIChatSession := &GenAIChatSessionMock{
//			SendMessageFunc: func(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
//				panic("mock out the SendMessage method")
//			},
//		}
//
//		// use mockedGenAIChatSession in code that requires interfaces.GenAIChatSession
//		// and then make assertions.
//
//	}
type GenAIChatSessionMock struct {
	// SendMessageFunc mocks the SendMessage method.
	SendMessageFunc func(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)

	// calls tracks calls to the methods.
	calls struct {
		// SendMessage holds details about calls to the SendMessage method.
		SendMessage []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Msg is the msg argument value.
			Msg []genai.Part
		}
	}
	lockSendMessage sync.RWMutex
}

// SendMessage calls SendMessageFunc.
func (mock *GenAIChatSessionMock) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	if mock.SendMessageFunc == nil {
		panic("GenAIChatSessionMock.SendMessageFunc: method is nil but GenAIChatSession.SendMessage was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Msg []genai.Part
	}{
		Ctx: ctx,
		Msg: msg,
	}
	mock.lockSendMessage.Lock()
	mock.calls.SendMessage = append(mock.calls.SendMessage, callInfo)
	mock.lockSendMessage.Unlock()
	return mock.SendMessageFunc(ctx, msg...)
}

// SendMessageCalls gets all the calls that were made to SendMessage.
// Check the length with:
//
//	len(mockedGenAIChatSession.SendMessageCalls())
func (mock *GenAIChatSessionMock) SendMessageCalls() []struct {
	Ctx context.Context
	Msg []genai.Part
} {
	var calls []struct {
		Ctx context.Context
		Msg []genai.Part
	}
	mock.lockSendMessage.RLock()
	calls = mock.calls.SendMessage
	mock.lockSendMessage.RUnlock()
	return calls
}

// Ensure, that PolicyClientMock does implement interfaces.PolicyClient.
// If this is not the case, regenerate this file with moq.
var _ interfaces.PolicyClient = &PolicyClientMock{}

// PolicyClientMock is a mock implementation of interfaces.PolicyClient.
//
//	func TestSomethingThatUsesPolicyClient(t *testing.T) {
//
//		// make and configure a mocked interfaces.PolicyClient
//		mockedPolicyClient := &PolicyClientMock{
//			QueryFunc: func(contextMoqParam context.Context, s string, v1 any, v2 any, queryOptions ...opac.QueryOption) error {
//				panic("mock out the Query method")
//			},
//		}
//
//		// use mockedPolicyClient in code that requires interfaces.PolicyClient
//		// and then make assertions.
//
//	}
type PolicyClientMock struct {
	// QueryFunc mocks the Query method.
	QueryFunc func(contextMoqParam context.Context, s string, v1 any, v2 any, queryOptions ...opac.QueryOption) error

	// calls tracks calls to the methods.
	calls struct {
		// Query holds details about calls to the Query method.
		Query []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
			// S is the s argument value.
			S string
			// V1 is the v1 argument value.
			V1 any
			// V2 is the v2 argument value.
			V2 any
			// QueryOptions is the queryOptions argument value.
			QueryOptions []opac.QueryOption
		}
	}
	lockQuery sync.RWMutex
}

// Query calls QueryFunc.
func (mock *PolicyClientMock) Query(contextMoqParam context.Context, s string, v1 any, v2 any, queryOptions ...opac.QueryOption) error {
	if mock.QueryFunc == nil {
		panic("PolicyClientMock.QueryFunc: method is nil but PolicyClient.Query was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
		S               string
		V1              any
		V2              any
		QueryOptions    []opac.QueryOption
	}{
		ContextMoqParam: contextMoqParam,
		S:               s,
		V1:              v1,
		V2:              v2,
		QueryOptions:    queryOptions,
	}
	mock.lockQuery.Lock()
	mock.calls.Query = append(mock.calls.Query, callInfo)
	mock.lockQuery.Unlock()
	return mock.QueryFunc(contextMoqParam, s, v1, v2, queryOptions...)
}

// QueryCalls gets all the calls that were made to Query.
// Check the length with:
//
//	len(mockedPolicyClient.QueryCalls())
func (mock *PolicyClientMock) QueryCalls() []struct {
	ContextMoqParam context.Context
	S               string
	V1              any
	V2              any
	QueryOptions    []opac.QueryOption
} {
	var calls []struct {
		ContextMoqParam context.Context
		S               string
		V1              any
		V2              any
		QueryOptions    []opac.QueryOption
	}
	mock.lockQuery.RLock()
	calls = mock.calls.Query
	mock.lockQuery.RUnlock()
	return calls
}

// Ensure, that RepositoryMock does implement interfaces.Repository.
// If this is not the case, regenerate this file with moq.
var _ interfaces.Repository = &RepositoryMock{}

// RepositoryMock is a mock implementation of interfaces.Repository.
//
//	func TestSomethingThatUsesRepository(t *testing.T) {
//
//		// make and configure a mocked interfaces.Repository
//		mockedRepository := &RepositoryMock{
//			FetchLatestAlertsFunc: func(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error) {
//				panic("mock out the FetchLatestAlerts method")
//			},
//			GetAlertFunc: func(ctx context.Context, alertID model.AlertID) (*model.Alert, error) {
//				panic("mock out the GetAlert method")
//			},
//			PutAlertFunc: func(ctx context.Context, alert model.Alert) error {
//				panic("mock out the PutAlert method")
//			},
//		}
//
//		// use mockedRepository in code that requires interfaces.Repository
//		// and then make assertions.
//
//	}
type RepositoryMock struct {
	// FetchLatestAlertsFunc mocks the FetchLatestAlerts method.
	FetchLatestAlertsFunc func(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error)

	// GetAlertFunc mocks the GetAlert method.
	GetAlertFunc func(ctx context.Context, alertID model.AlertID) (*model.Alert, error)

	// PutAlertFunc mocks the PutAlert method.
	PutAlertFunc func(ctx context.Context, alert model.Alert) error

	// calls tracks calls to the methods.
	calls struct {
		// FetchLatestAlerts holds details about calls to the FetchLatestAlerts method.
		FetchLatestAlerts []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Oldest is the oldest argument value.
			Oldest time.Time
			// Limit is the limit argument value.
			Limit int
		}
		// GetAlert holds details about calls to the GetAlert method.
		GetAlert []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// AlertID is the alertID argument value.
			AlertID model.AlertID
		}
		// PutAlert holds details about calls to the PutAlert method.
		PutAlert []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Alert is the alert argument value.
			Alert model.Alert
		}
	}
	lockFetchLatestAlerts sync.RWMutex
	lockGetAlert          sync.RWMutex
	lockPutAlert          sync.RWMutex
}

// FetchLatestAlerts calls FetchLatestAlertsFunc.
func (mock *RepositoryMock) FetchLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error) {
	if mock.FetchLatestAlertsFunc == nil {
		panic("RepositoryMock.FetchLatestAlertsFunc: method is nil but Repository.FetchLatestAlerts was just called")
	}
	callInfo := struct {
		Ctx    context.Context
		Oldest time.Time
		Limit  int
	}{
		Ctx:    ctx,
		Oldest: oldest,
		Limit:  limit,
	}
	mock.lockFetchLatestAlerts.Lock()
	mock.calls.FetchLatestAlerts = append(mock.calls.FetchLatestAlerts, callInfo)
	mock.lockFetchLatestAlerts.Unlock()
	return mock.FetchLatestAlertsFunc(ctx, oldest, limit)
}

// FetchLatestAlertsCalls gets all the calls that were made to FetchLatestAlerts.
// Check the length with:
//
//	len(mockedRepository.FetchLatestAlertsCalls())
func (mock *RepositoryMock) FetchLatestAlertsCalls() []struct {
	Ctx    context.Context
	Oldest time.Time
	Limit  int
} {
	var calls []struct {
		Ctx    context.Context
		Oldest time.Time
		Limit  int
	}
	mock.lockFetchLatestAlerts.RLock()
	calls = mock.calls.FetchLatestAlerts
	mock.lockFetchLatestAlerts.RUnlock()
	return calls
}

// GetAlert calls GetAlertFunc.
func (mock *RepositoryMock) GetAlert(ctx context.Context, alertID model.AlertID) (*model.Alert, error) {
	if mock.GetAlertFunc == nil {
		panic("RepositoryMock.GetAlertFunc: method is nil but Repository.GetAlert was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		AlertID model.AlertID
	}{
		Ctx:     ctx,
		AlertID: alertID,
	}
	mock.lockGetAlert.Lock()
	mock.calls.GetAlert = append(mock.calls.GetAlert, callInfo)
	mock.lockGetAlert.Unlock()
	return mock.GetAlertFunc(ctx, alertID)
}

// GetAlertCalls gets all the calls that were made to GetAlert.
// Check the length with:
//
//	len(mockedRepository.GetAlertCalls())
func (mock *RepositoryMock) GetAlertCalls() []struct {
	Ctx     context.Context
	AlertID model.AlertID
} {
	var calls []struct {
		Ctx     context.Context
		AlertID model.AlertID
	}
	mock.lockGetAlert.RLock()
	calls = mock.calls.GetAlert
	mock.lockGetAlert.RUnlock()
	return calls
}

// PutAlert calls PutAlertFunc.
func (mock *RepositoryMock) PutAlert(ctx context.Context, alert model.Alert) error {
	if mock.PutAlertFunc == nil {
		panic("RepositoryMock.PutAlertFunc: method is nil but Repository.PutAlert was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Alert model.Alert
	}{
		Ctx:   ctx,
		Alert: alert,
	}
	mock.lockPutAlert.Lock()
	mock.calls.PutAlert = append(mock.calls.PutAlert, callInfo)
	mock.lockPutAlert.Unlock()
	return mock.PutAlertFunc(ctx, alert)
}

// PutAlertCalls gets all the calls that were made to PutAlert.
// Check the length with:
//
//	len(mockedRepository.PutAlertCalls())
func (mock *RepositoryMock) PutAlertCalls() []struct {
	Ctx   context.Context
	Alert model.Alert
} {
	var calls []struct {
		Ctx   context.Context
		Alert model.Alert
	}
	mock.lockPutAlert.RLock()
	calls = mock.calls.PutAlert
	mock.lockPutAlert.RUnlock()
	return calls
}

// Ensure, that ActionMock does implement interfaces.Action.
// If this is not the case, regenerate this file with moq.
var _ interfaces.Action = &ActionMock{}

// ActionMock is a mock implementation of interfaces.Action.
//
//	func TestSomethingThatUsesAction(t *testing.T) {
//
//		// make and configure a mocked interfaces.Action
//		mockedAction := &ActionMock{
//			ExecuteFunc: func(ctx context.Context, ssn interfaces.GenAIChatSession, args model.Arguments) (string, error) {
//				panic("mock out the Execute method")
//			},
//			SpecFunc: func() model.ActionSpec {
//				panic("mock out the Spec method")
//			},
//		}
//
//		// use mockedAction in code that requires interfaces.Action
//		// and then make assertions.
//
//	}
type ActionMock struct {
	// ExecuteFunc mocks the Execute method.
	ExecuteFunc func(ctx context.Context, ssn interfaces.GenAIChatSession, args model.Arguments) (string, error)

	// SpecFunc mocks the Spec method.
	SpecFunc func() model.ActionSpec

	// calls tracks calls to the methods.
	calls struct {
		// Execute holds details about calls to the Execute method.
		Execute []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Ssn is the ssn argument value.
			Ssn interfaces.GenAIChatSession
			// Args is the args argument value.
			Args model.Arguments
		}
		// Spec holds details about calls to the Spec method.
		Spec []struct {
		}
	}
	lockExecute sync.RWMutex
	lockSpec    sync.RWMutex
}

// Execute calls ExecuteFunc.
func (mock *ActionMock) Execute(ctx context.Context, ssn interfaces.GenAIChatSession, args model.Arguments) (string, error) {
	if mock.ExecuteFunc == nil {
		panic("ActionMock.ExecuteFunc: method is nil but Action.Execute was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Ssn  interfaces.GenAIChatSession
		Args model.Arguments
	}{
		Ctx:  ctx,
		Ssn:  ssn,
		Args: args,
	}
	mock.lockExecute.Lock()
	mock.calls.Execute = append(mock.calls.Execute, callInfo)
	mock.lockExecute.Unlock()
	return mock.ExecuteFunc(ctx, ssn, args)
}

// ExecuteCalls gets all the calls that were made to Execute.
// Check the length with:
//
//	len(mockedAction.ExecuteCalls())
func (mock *ActionMock) ExecuteCalls() []struct {
	Ctx  context.Context
	Ssn  interfaces.GenAIChatSession
	Args model.Arguments
} {
	var calls []struct {
		Ctx  context.Context
		Ssn  interfaces.GenAIChatSession
		Args model.Arguments
	}
	mock.lockExecute.RLock()
	calls = mock.calls.Execute
	mock.lockExecute.RUnlock()
	return calls
}

// Spec calls SpecFunc.
func (mock *ActionMock) Spec() model.ActionSpec {
	if mock.SpecFunc == nil {
		panic("ActionMock.SpecFunc: method is nil but Action.Spec was just called")
	}
	callInfo := struct {
	}{}
	mock.lockSpec.Lock()
	mock.calls.Spec = append(mock.calls.Spec, callInfo)
	mock.lockSpec.Unlock()
	return mock.SpecFunc()
}

// SpecCalls gets all the calls that were made to Spec.
// Check the length with:
//
//	len(mockedAction.SpecCalls())
func (mock *ActionMock) SpecCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockSpec.RLock()
	calls = mock.calls.Spec
	mock.lockSpec.RUnlock()
	return calls
}
