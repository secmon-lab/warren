// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package githubapp_test

import (
	"context"
	"github.com/google/go-github/v69/github"
	"sync"
)

// ClientMock is a mock implementation of githubapp.Client.
//
//	func TestSomethingThatUsesClient(t *testing.T) {
//
//		// make and configure a mocked githubapp.Client
//		mockedClient := &ClientMock{
//			CommitChangesFunc: func(ctx context.Context, owner string, repo string, branch string, files map[string][]byte, message string) error {
//				panic("mock out the CommitChanges method")
//			},
//			CreateBranchFunc: func(ctx context.Context, owner string, repo string, baseBranch string, newBranch string) error {
//				panic("mock out the CreateBranch method")
//			},
//			CreatePullRequestFunc: func(ctx context.Context, owner string, repo string, title string, body string, head string, base string) (*github.PullRequest, error) {
//				panic("mock out the CreatePullRequest method")
//			},
//			GetDefaultBranchFunc: func(ctx context.Context, owner string, repo string) (string, error) {
//				panic("mock out the GetDefaultBranch method")
//			},
//			LookupBranchFunc: func(ctx context.Context, owner string, repo string, branch string) (*github.Reference, error) {
//				panic("mock out the LookupBranch method")
//			},
//		}
//
//		// use mockedClient in code that requires githubapp.Client
//		// and then make assertions.
//
//	}
type ClientMock struct {
	// CommitChangesFunc mocks the CommitChanges method.
	CommitChangesFunc func(ctx context.Context, owner string, repo string, branch string, files map[string][]byte, message string) error

	// CreateBranchFunc mocks the CreateBranch method.
	CreateBranchFunc func(ctx context.Context, owner string, repo string, baseBranch string, newBranch string) error

	// CreatePullRequestFunc mocks the CreatePullRequest method.
	CreatePullRequestFunc func(ctx context.Context, owner string, repo string, title string, body string, head string, base string) (*github.PullRequest, error)

	// GetDefaultBranchFunc mocks the GetDefaultBranch method.
	GetDefaultBranchFunc func(ctx context.Context, owner string, repo string) (string, error)

	// LookupBranchFunc mocks the LookupBranch method.
	LookupBranchFunc func(ctx context.Context, owner string, repo string, branch string) (*github.Reference, error)

	// calls tracks calls to the methods.
	calls struct {
		// CommitChanges holds details about calls to the CommitChanges method.
		CommitChanges []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Owner is the owner argument value.
			Owner string
			// Repo is the repo argument value.
			Repo string
			// Branch is the branch argument value.
			Branch string
			// Files is the files argument value.
			Files map[string][]byte
			// Message is the message argument value.
			Message string
		}
		// CreateBranch holds details about calls to the CreateBranch method.
		CreateBranch []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Owner is the owner argument value.
			Owner string
			// Repo is the repo argument value.
			Repo string
			// BaseBranch is the baseBranch argument value.
			BaseBranch string
			// NewBranch is the newBranch argument value.
			NewBranch string
		}
		// CreatePullRequest holds details about calls to the CreatePullRequest method.
		CreatePullRequest []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Owner is the owner argument value.
			Owner string
			// Repo is the repo argument value.
			Repo string
			// Title is the title argument value.
			Title string
			// Body is the body argument value.
			Body string
			// Head is the head argument value.
			Head string
			// Base is the base argument value.
			Base string
		}
		// GetDefaultBranch holds details about calls to the GetDefaultBranch method.
		GetDefaultBranch []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Owner is the owner argument value.
			Owner string
			// Repo is the repo argument value.
			Repo string
		}
		// LookupBranch holds details about calls to the LookupBranch method.
		LookupBranch []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Owner is the owner argument value.
			Owner string
			// Repo is the repo argument value.
			Repo string
			// Branch is the branch argument value.
			Branch string
		}
	}
	lockCommitChanges     sync.RWMutex
	lockCreateBranch      sync.RWMutex
	lockCreatePullRequest sync.RWMutex
	lockGetDefaultBranch  sync.RWMutex
	lockLookupBranch      sync.RWMutex
}

// CommitChanges calls CommitChangesFunc.
func (mock *ClientMock) CommitChanges(ctx context.Context, owner string, repo string, branch string, files map[string][]byte, message string) error {
	if mock.CommitChangesFunc == nil {
		panic("ClientMock.CommitChangesFunc: method is nil but Client.CommitChanges was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		Owner   string
		Repo    string
		Branch  string
		Files   map[string][]byte
		Message string
	}{
		Ctx:     ctx,
		Owner:   owner,
		Repo:    repo,
		Branch:  branch,
		Files:   files,
		Message: message,
	}
	mock.lockCommitChanges.Lock()
	mock.calls.CommitChanges = append(mock.calls.CommitChanges, callInfo)
	mock.lockCommitChanges.Unlock()
	return mock.CommitChangesFunc(ctx, owner, repo, branch, files, message)
}

// CommitChangesCalls gets all the calls that were made to CommitChanges.
// Check the length with:
//
//	len(mockedClient.CommitChangesCalls())
func (mock *ClientMock) CommitChangesCalls() []struct {
	Ctx     context.Context
	Owner   string
	Repo    string
	Branch  string
	Files   map[string][]byte
	Message string
} {
	var calls []struct {
		Ctx     context.Context
		Owner   string
		Repo    string
		Branch  string
		Files   map[string][]byte
		Message string
	}
	mock.lockCommitChanges.RLock()
	calls = mock.calls.CommitChanges
	mock.lockCommitChanges.RUnlock()
	return calls
}

// CreateBranch calls CreateBranchFunc.
func (mock *ClientMock) CreateBranch(ctx context.Context, owner string, repo string, baseBranch string, newBranch string) error {
	if mock.CreateBranchFunc == nil {
		panic("ClientMock.CreateBranchFunc: method is nil but Client.CreateBranch was just called")
	}
	callInfo := struct {
		Ctx        context.Context
		Owner      string
		Repo       string
		BaseBranch string
		NewBranch  string
	}{
		Ctx:        ctx,
		Owner:      owner,
		Repo:       repo,
		BaseBranch: baseBranch,
		NewBranch:  newBranch,
	}
	mock.lockCreateBranch.Lock()
	mock.calls.CreateBranch = append(mock.calls.CreateBranch, callInfo)
	mock.lockCreateBranch.Unlock()
	return mock.CreateBranchFunc(ctx, owner, repo, baseBranch, newBranch)
}

// CreateBranchCalls gets all the calls that were made to CreateBranch.
// Check the length with:
//
//	len(mockedClient.CreateBranchCalls())
func (mock *ClientMock) CreateBranchCalls() []struct {
	Ctx        context.Context
	Owner      string
	Repo       string
	BaseBranch string
	NewBranch  string
} {
	var calls []struct {
		Ctx        context.Context
		Owner      string
		Repo       string
		BaseBranch string
		NewBranch  string
	}
	mock.lockCreateBranch.RLock()
	calls = mock.calls.CreateBranch
	mock.lockCreateBranch.RUnlock()
	return calls
}

// CreatePullRequest calls CreatePullRequestFunc.
func (mock *ClientMock) CreatePullRequest(ctx context.Context, owner string, repo string, title string, body string, head string, base string) (*github.PullRequest, error) {
	if mock.CreatePullRequestFunc == nil {
		panic("ClientMock.CreatePullRequestFunc: method is nil but Client.CreatePullRequest was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Owner string
		Repo  string
		Title string
		Body  string
		Head  string
		Base  string
	}{
		Ctx:   ctx,
		Owner: owner,
		Repo:  repo,
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}
	mock.lockCreatePullRequest.Lock()
	mock.calls.CreatePullRequest = append(mock.calls.CreatePullRequest, callInfo)
	mock.lockCreatePullRequest.Unlock()
	return mock.CreatePullRequestFunc(ctx, owner, repo, title, body, head, base)
}

// CreatePullRequestCalls gets all the calls that were made to CreatePullRequest.
// Check the length with:
//
//	len(mockedClient.CreatePullRequestCalls())
func (mock *ClientMock) CreatePullRequestCalls() []struct {
	Ctx   context.Context
	Owner string
	Repo  string
	Title string
	Body  string
	Head  string
	Base  string
} {
	var calls []struct {
		Ctx   context.Context
		Owner string
		Repo  string
		Title string
		Body  string
		Head  string
		Base  string
	}
	mock.lockCreatePullRequest.RLock()
	calls = mock.calls.CreatePullRequest
	mock.lockCreatePullRequest.RUnlock()
	return calls
}

// GetDefaultBranch calls GetDefaultBranchFunc.
func (mock *ClientMock) GetDefaultBranch(ctx context.Context, owner string, repo string) (string, error) {
	if mock.GetDefaultBranchFunc == nil {
		panic("ClientMock.GetDefaultBranchFunc: method is nil but Client.GetDefaultBranch was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Owner string
		Repo  string
	}{
		Ctx:   ctx,
		Owner: owner,
		Repo:  repo,
	}
	mock.lockGetDefaultBranch.Lock()
	mock.calls.GetDefaultBranch = append(mock.calls.GetDefaultBranch, callInfo)
	mock.lockGetDefaultBranch.Unlock()
	return mock.GetDefaultBranchFunc(ctx, owner, repo)
}

// GetDefaultBranchCalls gets all the calls that were made to GetDefaultBranch.
// Check the length with:
//
//	len(mockedClient.GetDefaultBranchCalls())
func (mock *ClientMock) GetDefaultBranchCalls() []struct {
	Ctx   context.Context
	Owner string
	Repo  string
} {
	var calls []struct {
		Ctx   context.Context
		Owner string
		Repo  string
	}
	mock.lockGetDefaultBranch.RLock()
	calls = mock.calls.GetDefaultBranch
	mock.lockGetDefaultBranch.RUnlock()
	return calls
}

// LookupBranch calls LookupBranchFunc.
func (mock *ClientMock) LookupBranch(ctx context.Context, owner string, repo string, branch string) (*github.Reference, error) {
	if mock.LookupBranchFunc == nil {
		panic("ClientMock.LookupBranchFunc: method is nil but Client.LookupBranch was just called")
	}
	callInfo := struct {
		Ctx    context.Context
		Owner  string
		Repo   string
		Branch string
	}{
		Ctx:    ctx,
		Owner:  owner,
		Repo:   repo,
		Branch: branch,
	}
	mock.lockLookupBranch.Lock()
	mock.calls.LookupBranch = append(mock.calls.LookupBranch, callInfo)
	mock.lockLookupBranch.Unlock()
	return mock.LookupBranchFunc(ctx, owner, repo, branch)
}

// LookupBranchCalls gets all the calls that were made to LookupBranch.
// Check the length with:
//
//	len(mockedClient.LookupBranchCalls())
func (mock *ClientMock) LookupBranchCalls() []struct {
	Ctx    context.Context
	Owner  string
	Repo   string
	Branch string
} {
	var calls []struct {
		Ctx    context.Context
		Owner  string
		Repo   string
		Branch string
	}
	mock.lockLookupBranch.RLock()
	calls = mock.calls.LookupBranch
	mock.lockLookupBranch.RUnlock()
	return calls
}
