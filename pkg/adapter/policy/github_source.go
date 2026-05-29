package policy

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/utils/async"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// DefaultGitHubCacheTTL is the default time-to-live for GitHubSource cache.
const DefaultGitHubCacheTTL = time.Minute

// GitHubSourceOpts configures a GitHubSource.
//
// Either Client must be provided directly (typically for testing), or all of
// AppID, InstallationID, and PrivateKey must be provided so that a GitHub App
// installation token can be obtained.
type GitHubSourceOpts struct {
	Owner string
	Repo  string

	// Paths within the repository to recursively scan for .rego files.
	// An empty slice scans the repository root.
	Paths []string

	// TTL for the in-memory cache. Zero means DefaultGitHubCacheTTL.
	TTL time.Duration

	// Optional pre-built client. When non-nil, App credentials below are ignored.
	Client *github.Client

	// GitHub App credentials. Required when Client is nil.
	AppID          int64
	InstallationID int64
	PrivateKey     []byte
}

// GitHubSource fetches Rego policy files from a GitHub repository's default
// branch HEAD. Results are cached in memory for TTL; sha-based change detection
// avoids re-fetching content when the branch HEAD has not moved.
//
// Once a snapshot has been cached, Snapshot follows a stale-while-revalidate
// model: an expired cache is served immediately while a single background
// refresh runs, so callers (e.g. the alert pipeline) never block on a slow or
// failing GitHub fetch. A failed refresh keeps the previous snapshot and is
// reported via errutil.Handle. Only the very first load (no cache yet) fetches
// synchronously and propagates its error, since there is nothing to serve.
type GitHubSource struct {
	owner  string
	repo   string
	paths  []string
	ttl    time.Duration
	client *github.Client

	mu          sync.Mutex
	cachedFiles map[string]string
	cachedSha   string
	cachedAt    time.Time
	// refreshing guarantees at most one background refresh runs at a time. It is
	// an atomic so the check-and-set is a single atomic operation independent of
	// s.mu (which protects the cached* fields).
	refreshing atomic.Bool
	// refreshWG tracks in-flight background refreshes so tests can await them
	// deterministically; it has no effect on production behaviour.
	refreshWG sync.WaitGroup
}

// NewGitHubSource constructs a GitHubSource. If opts.Client is nil, App
// credentials are required and a github.Client is built using
// bradleyfalzon/ghinstallation/v2.
func NewGitHubSource(opts GitHubSourceOpts) (*GitHubSource, error) {
	if opts.Owner == "" || opts.Repo == "" {
		return nil, goerr.New("owner and repo are required")
	}

	client := opts.Client
	if client == nil {
		if opts.AppID == 0 || opts.InstallationID == 0 || len(opts.PrivateKey) == 0 {
			return nil, goerr.New("GitHub App credentials are required when Client is not provided")
		}
		transport, err := ghinstallation.New(http.DefaultTransport, opts.AppID, opts.InstallationID, opts.PrivateKey)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create GitHub App transport")
		}
		client = github.NewClient(&http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		})
	}

	paths := opts.Paths
	if len(paths) == 0 {
		paths = []string{""}
	}

	ttl := opts.TTL
	if ttl <= 0 {
		ttl = DefaultGitHubCacheTTL
	}

	return &GitHubSource{
		owner:  opts.Owner,
		repo:   opts.Repo,
		paths:  paths,
		ttl:    ttl,
		client: client,
	}, nil
}

// Snapshot returns the current snapshot. When a cached snapshot exists it is
// returned without blocking: a fresh cache is returned directly, and an expired
// cache is returned as-is while a single background refresh is triggered
// (stale-while-revalidate). Only the first load, when no cache exists yet,
// fetches synchronously and may return an error.
func (s *GitHubSource) Snapshot(ctx context.Context) (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedFiles != nil {
		if time.Since(s.cachedAt) < s.ttl {
			return s.snapshotLocked(), nil
		}
		// Cache is stale: refresh in the background and serve the existing
		// snapshot immediately so the caller never blocks on GitHub.
		s.triggerRefresh(ctx)
		return s.snapshotLocked(), nil
	}

	// First load: there is no cached snapshot to fall back to, so fetch
	// synchronously and propagate any error to the caller (e.g. Prime fails
	// at startup and the process exits before the HTTP server starts).
	files, sha, err := s.fetchRemote(ctx, "")
	if err != nil {
		return nil, err
	}
	s.cachedFiles = files
	s.cachedSha = sha
	s.cachedAt = time.Now()
	return s.snapshotLocked(), nil
}

func (s *GitHubSource) snapshotLocked() *Snapshot {
	return &Snapshot{
		Files:   s.cachedFiles,
		Version: "github://" + s.owner + "/" + s.repo + "@" + s.cachedSha,
	}
}

// triggerRefresh starts a single background refresh unless one is already
// running. The single-flight guard is a CompareAndSwap on s.refreshing, so the
// check-and-set is atomic and does not depend on holding s.mu. The refresh runs
// via async.Dispatch, which detaches the request context (so the refresh is not
// cancelled when the originating request completes) and reports a returned error
// via errutil.Handle. The guard is always released via the deferred Store(false),
// even if the refresh returns early or panics, so a failed refresh can never
// permanently disable future refreshes.
func (s *GitHubSource) triggerRefresh(ctx context.Context) {
	if !s.refreshing.CompareAndSwap(false, true) {
		return // a refresh is already in flight
	}
	s.refreshWG.Add(1)
	async.Dispatch(ctx, func(ctx context.Context) error {
		defer s.refreshWG.Done()
		defer s.refreshing.Store(false)
		return s.runRefresh(ctx)
	})
}

// runRefresh performs a background refresh without holding the mutex during the
// network fetch, then swaps in the result under a brief lock. On any failure it
// keeps the previous snapshot and returns the error (reported by async.Dispatch
// via errutil.Handle). cachedAt is stamped on both success and failure so a
// failing GitHub backs off for a full TTL instead of being retried on every
// alert. The single-flight guard (s.refreshing) is released by the caller via
// the deferred Store(false).
func (s *GitHubSource) runRefresh(ctx context.Context) error {
	s.mu.Lock()
	prevSha := s.cachedSha
	s.mu.Unlock()

	files, sha, fetchErr := s.fetchRemote(ctx, prevSha)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedAt = time.Now()
	if fetchErr != nil {
		return fetchErr
	}
	if files != nil { // nil files means the HEAD sha was unchanged.
		s.cachedFiles = files
		s.cachedSha = sha
	}
	return nil
}

// fetchRemote fetches the policy files from GitHub without touching the cache or
// holding the mutex, so concurrent readers are never blocked by a slow fetch.
// It returns (nil, headSha, nil) when prevSha matches the current HEAD (no
// change). On any failure it returns a wrapped error and leaves the cache
// untouched for the caller to preserve.
func (s *GitHubSource) fetchRemote(ctx context.Context, prevSha string) (map[string]string, string, error) {
	repoInfo, _, err := s.client.Repositories.Get(ctx, s.owner, s.repo)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to fetch repo info",
			goerr.T(errutil.TagGitHubError),
			goerr.V("owner", s.owner), goerr.V("repo", s.repo))
	}
	defaultBranch := repoInfo.GetDefaultBranch()
	if defaultBranch == "" {
		return nil, "", goerr.New("repository has no default branch",
			goerr.T(errutil.TagGitHubError),
			goerr.V("owner", s.owner), goerr.V("repo", s.repo))
	}

	branch, _, err := s.client.Repositories.GetBranch(ctx, s.owner, s.repo, defaultBranch, 0)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to fetch branch",
			goerr.T(errutil.TagGitHubError),
			goerr.V("branch", defaultBranch))
	}
	headSha := branch.GetCommit().GetSHA()
	if headSha == "" {
		return nil, "", goerr.New("branch has no HEAD commit",
			goerr.T(errutil.TagGitHubError),
			goerr.V("branch", defaultBranch))
	}

	if prevSha != "" && headSha == prevSha {
		return nil, headSha, nil
	}

	files, err := s.fetchAllPaths(ctx, headSha)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to fetch policy files",
			goerr.T(errutil.TagGitHubError),
			goerr.V("commit_sha", headSha))
	}

	if err := validatePolicy(ctx, files); err != nil {
		return nil, "", goerr.Wrap(err, "policy validation failed",
			goerr.T(errutil.TagValidation),
			goerr.V("commit_sha", headSha))
	}

	return files, headSha, nil
}

func (s *GitHubSource) fetchAllPaths(ctx context.Context, sha string) (map[string]string, error) {
	out := map[string]string{}
	opts := &github.RepositoryContentGetOptions{Ref: sha}
	for _, p := range s.paths {
		if err := s.fetchPath(ctx, p, opts, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *GitHubSource) fetchPath(ctx context.Context, p string, opts *github.RepositoryContentGetOptions, out map[string]string) error {
	fileContent, dirContents, _, err := s.client.Repositories.GetContents(ctx, s.owner, s.repo, p, opts)
	if err != nil {
		return goerr.Wrap(err, "failed to fetch contents", goerr.V("path", p))
	}

	if fileContent != nil {
		if !strings.HasSuffix(fileContent.GetName(), ".rego") {
			return nil
		}
		content, err := fileContent.GetContent()
		if err != nil {
			return goerr.Wrap(err, "failed to decode file content",
				goerr.V("path", fileContent.GetPath()))
		}
		out[s.fileKey(fileContent.GetPath())] = content
		return nil
	}

	for _, entry := range dirContents {
		switch entry.GetType() {
		case "file":
			if !strings.HasSuffix(entry.GetName(), ".rego") {
				continue
			}
			if err := s.fetchPath(ctx, entry.GetPath(), opts, out); err != nil {
				return err
			}
		case "dir":
			if err := s.fetchPath(ctx, entry.GetPath(), opts, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *GitHubSource) fileKey(repoPath string) string {
	return "github://" + s.owner + "/" + s.repo + "/" + repoPath
}

// validatePolicy performs both compile-time and runtime checks against a set
// of Rego files. The compile step catches syntax and reference errors; the
// runtime step (evaluating the root data document with an empty input)
// surfaces problems such as complete-rule conflicts that pass compilation
// but fail at evaluation time. ErrNoEvalResult is treated as success since
// it merely indicates that the requested document is not defined.
func validatePolicy(ctx context.Context, files map[string]string) error {
	client, err := opaq.New(opaq.DataMap(files))
	if err != nil {
		return goerr.Wrap(err, "failed to compile policy")
	}
	var out any
	if err := client.Query(ctx, "data", map[string]any{}, &out); err != nil {
		if errors.Is(err, opaq.ErrNoEvalResult) {
			return nil
		}
		return goerr.Wrap(err, "policy evaluation failed under empty input")
	}
	return nil
}
