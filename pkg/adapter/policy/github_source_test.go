package policy_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/gt"
	policyadapter "github.com/secmon-lab/warren/pkg/adapter/policy"
)

type mockGitHub struct {
	t       *testing.T
	server  *httptest.Server
	mu      sync.Mutex
	headSha string
	// fileTree maps repo paths (e.g. "policies/x.rego") to rego contents.
	fileTree     map[string]string
	repoHits     int
	branchHits   int
	contentsHits int
}

func newMockGitHub(t *testing.T, headSha string, fileTree map[string]string) *mockGitHub {
	m := &mockGitHub{
		t:        t,
		headSha:  headSha,
		fileTree: fileTree,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo", m.handleRepo)
	mux.HandleFunc("/repos/owner/repo/branches/main", m.handleBranch)
	mux.HandleFunc("/repos/owner/repo/contents/", m.handleContents)
	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

func (m *mockGitHub) update(headSha string, fileTree map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headSha = headSha
	m.fileTree = fileTree
}

func (m *mockGitHub) client() *github.Client {
	u, err := url.Parse(m.server.URL + "/")
	gt.NoError(m.t, err)
	c := github.NewClient(nil)
	c.BaseURL = u
	c.UploadURL = u
	return c
}

func (m *mockGitHub) handleRepo(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.repoHits++
	m.mu.Unlock()
	_, _ = fmt.Fprintln(w, `{"default_branch": "main"}`)
}

func (m *mockGitHub) handleBranch(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.branchHits++
	sha := m.headSha
	m.mu.Unlock()
	_, _ = fmt.Fprintf(w, `{"name": "main", "commit": {"sha": %q}}`+"\n", sha)
}

func (m *mockGitHub) handleContents(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.contentsHits++
	tree := m.fileTree
	m.mu.Unlock()

	repoPath := strings.TrimPrefix(r.URL.Path, "/repos/owner/repo/contents/")
	repoPath = strings.TrimSuffix(repoPath, "/")

	if content, ok := tree[repoPath]; ok {
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		_, _ = fmt.Fprintf(w,
			`{"name": %q, "path": %q, "type": "file", "encoding": "base64", "content": %q}`+"\n",
			path.Base(repoPath), repoPath, encoded)
		return
	}

	prefix := repoPath
	if prefix != "" {
		prefix += "/"
	}

	var entries []string
	for filePath := range tree {
		if !strings.HasPrefix(filePath, prefix) {
			continue
		}
		rest := filePath[len(prefix):]
		if rest == "" || strings.Contains(rest, "/") {
			continue
		}
		entries = append(entries,
			fmt.Sprintf(`{"name": %q, "path": %q, "type": "file"}`, rest, filePath))
	}
	_, _ = fmt.Fprintf(w, "[%s]\n", strings.Join(entries, ","))
}

const ghTestRego = `package ingest.fromgh
import rego.v1
alerts contains a if {
	input.kind == "test"
	a := {"title": "from-github"}
}
`

const ghTestRegoV2 = `package ingest.fromgh
import rego.v1
alerts contains a if {
	input.kind == "test"
	a := {"title": "from-github-v2"}
}
`

func TestGitHubSource_InitialFetch(t *testing.T) {
	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Minute,
	})
	gt.NoError(t, err)

	snap, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.NotNil(t, snap)
	gt.M(t, snap.Files).Length(1)
	gt.True(t, strings.Contains(snap.Version, "sha-1"))

	gt.Equal(t, mock.repoHits, 1)
	gt.Equal(t, mock.branchHits, 1)
	gt.True(t, mock.contentsHits >= 1)
}

func TestGitHubSource_CachesWithinTTL(t *testing.T) {
	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Hour,
	})
	gt.NoError(t, err)

	_, err = src.Snapshot(context.Background())
	gt.NoError(t, err)
	repoBefore := mock.repoHits

	_, err = src.Snapshot(context.Background())
	gt.NoError(t, err)

	gt.Equal(t, mock.repoHits, repoBefore) // no additional API calls within TTL
}

func TestGitHubSource_RefreshesAfterTTL_SameSha(t *testing.T) {
	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Nanosecond, // immediately expire
	})
	gt.NoError(t, err)

	_, err = src.Snapshot(context.Background())
	gt.NoError(t, err)
	contentsBefore := mock.contentsHits

	// Wait a moment to ensure TTL window passes.
	time.Sleep(2 * time.Millisecond)

	// Stale cache is served immediately while a background refresh runs.
	snap, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.True(t, strings.Contains(snap.Version, "sha-1"))

	// Await the background refresh deterministically.
	src.WaitRefresh()

	// repo + branch are re-fetched; contents must NOT be re-fetched (sha unchanged).
	gt.Equal(t, mock.contentsHits, contentsBefore)
}

func TestGitHubSource_RefreshesAfterTTL_NewSha(t *testing.T) {
	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Nanosecond,
	})
	gt.NoError(t, err)

	first, err := src.Snapshot(context.Background())
	gt.NoError(t, err)

	mock.update("sha-2", map[string]string{
		"policies/ignore.rego": ghTestRegoV2,
	})

	time.Sleep(2 * time.Millisecond)

	// First post-expiry call serves the stale snapshot and triggers a refresh.
	second, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, first.Version, second.Version)

	src.WaitRefresh()

	// Once the refresh has applied, the new sha is visible.
	third, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.NotEqual(t, first.Version, third.Version)
	gt.True(t, strings.Contains(third.Version, "sha-2"))
	gt.True(t, strings.Contains(third.Files["github://owner/repo/policies/ignore.rego"], "from-github-v2"))

	// Drain the no-op refresh re-triggered by the read above.
	src.WaitRefresh()
}

func TestGitHubSource_ValidationFailure_FallbackToCached(t *testing.T) {
	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Nanosecond,
	})
	gt.NoError(t, err)

	first, err := src.Snapshot(context.Background())
	gt.NoError(t, err)

	// New sha but bad rego.
	mock.update("sha-broken", map[string]string{
		"policies/ignore.rego": "this is not rego",
	})

	time.Sleep(2 * time.Millisecond)

	// Stale snapshot is served immediately; the background refresh will fail.
	snap, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, snap.Version, first.Version)

	src.WaitRefresh()

	// The failed refresh keeps the previous good snapshot intact.
	snap2, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, snap2.Version, first.Version)
	gt.True(t, strings.Contains(snap2.Files["github://owner/repo/policies/ignore.rego"], "from-github\""))

	src.WaitRefresh()
}

func TestGitHubSource_RuntimeConflict_FallbackToCached(t *testing.T) {
	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Nanosecond,
	})
	gt.NoError(t, err)

	first, err := src.Snapshot(context.Background())
	gt.NoError(t, err)

	// Compile succeeds but evaluating data triggers a complete-rule conflict.
	mock.update("sha-conflict", map[string]string{
		"policies/conflict.rego": `package conflict
import rego.v1
x := 1
x := 2
`,
	})

	time.Sleep(2 * time.Millisecond)

	// Stale snapshot is served immediately; the background refresh will fail
	// the runtime conflict check.
	snap, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, snap.Version, first.Version)

	src.WaitRefresh()

	// The failed refresh keeps the previous good snapshot intact.
	snap2, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, snap2.Version, first.Version)
	gt.True(t, strings.Contains(snap2.Files["github://owner/repo/policies/ignore.rego"], "from-github\""))

	src.WaitRefresh()
}

func TestGitHubSource_RuntimeConflict_NoCache_ReturnsError(t *testing.T) {
	mock := newMockGitHub(t, "sha-conflict", map[string]string{
		"policies/conflict.rego": `package conflict
import rego.v1
x := 1
x := 2
`,
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Minute,
	})
	gt.NoError(t, err)

	_, err = src.Snapshot(context.Background())
	gt.Error(t, err)
}

func TestGitHubSource_ValidationFailure_NoCache_ReturnsError(t *testing.T) {
	mock := newMockGitHub(t, "sha-broken", map[string]string{
		"policies/ignore.rego": "not valid rego at all",
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Minute,
	})
	gt.NoError(t, err)

	_, err = src.Snapshot(context.Background())
	gt.Error(t, err)
}

func TestGitHubSource_ConcurrentSnapshot_SerialisesFetch(t *testing.T) {
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32

	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})
	// Wrap the original handler to detect concurrent fetches.
	origHandler := mock.server.Config.Handler
	mock.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := inFlight.Add(1)
		for {
			oldMax := maxInFlight.Load()
			if current <= oldMax {
				break
			}
			if maxInFlight.CompareAndSwap(oldMax, current) {
				break
			}
		}
		// Tiny delay to widen the race window.
		time.Sleep(5 * time.Millisecond)
		origHandler.ServeHTTP(w, r)
		inFlight.Add(-1)
	})

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: mock.client(),
		TTL:    time.Minute,
	})
	gt.NoError(t, err)

	const N = 10
	var wg sync.WaitGroup
	for range N {
		wg.Go(func() {
			_, err := src.Snapshot(context.Background())
			gt.NoError(t, err)
		})
	}
	wg.Wait()

	// Even with concurrent callers, the lock ensures fetch happens once;
	// after that, cache is hit. So total repo API hits should be 1.
	gt.Equal(t, mock.repoHits, 1)
	// And no concurrent fetches inside the lock window.
	gt.True(t, maxInFlight.Load() <= 1)
}

// blockingTransport delays every HTTP round trip until release is closed while
// blocked is set. It simulates a slow or hung GitHub on the client side, which
// is fully thread-safe (unlike mutating the test server's handler at runtime).
type blockingTransport struct {
	blocked *atomic.Bool
	release chan struct{}
	base    http.RoundTripper
}

func (t *blockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.blocked.Load() {
		<-t.release
	}
	return t.base.RoundTrip(req)
}

func TestGitHubSource_StaleServedImmediately_NonBlocking(t *testing.T) {
	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})

	u, err := url.Parse(mock.server.URL + "/")
	gt.NoError(t, err)

	release := make(chan struct{})
	var blocked atomic.Bool
	gClient := github.NewClient(&http.Client{
		Transport: &blockingTransport{
			blocked: &blocked,
			release: release,
			base:    http.DefaultTransport,
		},
	})
	gClient.BaseURL = u
	gClient.UploadURL = u

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: gClient,
		TTL:    time.Nanosecond, // immediately expire after the first load
	})
	gt.NoError(t, err)

	// Warm the cache with a fast fetch.
	first, err := src.Snapshot(context.Background())
	gt.NoError(t, err)

	// From now on, every GitHub call blocks until released, simulating a slow
	// or hung GitHub. The background refresh will be stuck in RoundTrip.
	blocked.Store(true)

	// Let the TTL window pass so the next Snapshot sees a stale cache.
	time.Sleep(2 * time.Millisecond)

	// Snapshot MUST return the stale snapshot immediately even though the
	// background refresh is blocked on the hung GitHub server. If Snapshot
	// blocked on the fetch, this test would deadlock (release is still open).
	snap, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, snap.Version, first.Version)

	// Release the blocked refresh and drain it.
	close(release)
	src.WaitRefresh()
}

// panicTransport panics on every round trip while shouldPanic is set, to
// simulate an unexpected failure inside the background refresh goroutine.
type panicTransport struct {
	shouldPanic *atomic.Bool
	base        http.RoundTripper
}

func (t *panicTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.shouldPanic.Load() {
		panic("simulated transport panic")
	}
	return t.base.RoundTrip(req)
}

func TestGitHubSource_RefreshPanic_DoesNotWedgeSingleFlight(t *testing.T) {
	mock := newMockGitHub(t, "sha-1", map[string]string{
		"policies/ignore.rego": ghTestRego,
	})

	u, err := url.Parse(mock.server.URL + "/")
	gt.NoError(t, err)

	var shouldPanic atomic.Bool
	gClient := github.NewClient(&http.Client{
		Transport: &panicTransport{
			shouldPanic: &shouldPanic,
			base:        http.DefaultTransport,
		},
	})
	gClient.BaseURL = u
	gClient.UploadURL = u

	src, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
		Owner:  "owner",
		Repo:   "repo",
		Paths:  []string{"policies"},
		Client: gClient,
		TTL:    time.Nanosecond, // immediately expire after the first load
	})
	gt.NoError(t, err)

	// Warm the cache with a fast fetch.
	first, err := src.Snapshot(context.Background())
	gt.NoError(t, err)

	// Make the next background refresh panic inside RoundTrip.
	shouldPanic.Store(true)
	mock.update("sha-2", map[string]string{
		"policies/ignore.rego": ghTestRegoV2,
	})
	time.Sleep(2 * time.Millisecond)

	// Stale snapshot is served; the triggered refresh panics in the background
	// and is recovered by async.Dispatch.
	snap, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, snap.Version, first.Version)
	src.WaitRefresh()

	// After the panic, the single-flight guard must have been released: a
	// subsequent (now healthy) refresh must run and apply the new sha. If the
	// guard were wedged at true, this refresh would be skipped and the version
	// would stay at sha-1.
	shouldPanic.Store(false)
	time.Sleep(2 * time.Millisecond)

	stale, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, stale.Version, first.Version) // still stale until refresh applies
	src.WaitRefresh()

	updated, err := src.Snapshot(context.Background())
	gt.NoError(t, err)
	gt.True(t, strings.Contains(updated.Version, "sha-2"))
	gt.True(t, strings.Contains(updated.Files["github://owner/repo/policies/ignore.rego"], "from-github-v2"))
	src.WaitRefresh()
}
