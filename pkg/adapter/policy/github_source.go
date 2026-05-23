package policy

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
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
// avoids re-fetching content when the branch HEAD has not moved. On any
// fetch or validation failure, the previously known good snapshot is returned.
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

// Snapshot returns the current snapshot, fetching from GitHub if the cache is
// stale. Concurrent callers serialise on the internal mutex so only one fetch
// is in flight at a time.
func (s *GitHubSource) Snapshot(ctx context.Context) (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedFiles != nil && time.Since(s.cachedAt) < s.ttl {
		return s.snapshotLocked(), nil
	}

	repoInfo, _, err := s.client.Repositories.Get(ctx, s.owner, s.repo)
	if err != nil {
		return s.fallbackLocked(ctx, goerr.Wrap(err, "failed to fetch repo info",
			goerr.V("owner", s.owner), goerr.V("repo", s.repo)))
	}
	defaultBranch := repoInfo.GetDefaultBranch()
	if defaultBranch == "" {
		return s.fallbackLocked(ctx, goerr.New("repository has no default branch",
			goerr.V("owner", s.owner), goerr.V("repo", s.repo)))
	}

	branch, _, err := s.client.Repositories.GetBranch(ctx, s.owner, s.repo, defaultBranch, 0)
	if err != nil {
		return s.fallbackLocked(ctx, goerr.Wrap(err, "failed to fetch branch",
			goerr.V("branch", defaultBranch)))
	}
	headSha := branch.GetCommit().GetSHA()
	if headSha == "" {
		return s.fallbackLocked(ctx, goerr.New("branch has no HEAD commit",
			goerr.V("branch", defaultBranch)))
	}

	if s.cachedFiles != nil && headSha == s.cachedSha {
		s.cachedAt = time.Now()
		return s.snapshotLocked(), nil
	}

	files, err := s.fetchAllPaths(ctx, headSha)
	if err != nil {
		return s.fallbackLocked(ctx, goerr.Wrap(err, "failed to fetch policy files",
			goerr.V("commit_sha", headSha)))
	}

	if err := validatePolicy(ctx, files); err != nil {
		// Validation failures indicate a data-level problem (bad Rego, rule
		// conflict, etc.) that will not self-heal on retry, so notify
		// regardless of whether a cached snapshot is available.
		wrapped := goerr.Wrap(err, "policy validation failed",
			goerr.V("commit_sha", headSha))
		errutil.Handle(ctx, wrapped)
		if s.cachedFiles != nil {
			return s.snapshotLocked(), nil
		}
		return nil, wrapped
	}

	s.cachedFiles = files
	s.cachedSha = headSha
	s.cachedAt = time.Now()
	return s.snapshotLocked(), nil
}

func (s *GitHubSource) snapshotLocked() *Snapshot {
	return &Snapshot{
		Files:   s.cachedFiles,
		Version: "github://" + s.owner + "/" + s.repo + "@" + s.cachedSha,
	}
}

// fallbackLocked decides what to return when a fetch step fails.
// If a previous snapshot exists, the error is logged via errutil.Handle and
// the cached snapshot is returned. Otherwise the error is returned.
func (s *GitHubSource) fallbackLocked(ctx context.Context, err error) (*Snapshot, error) {
	if s.cachedFiles != nil {
		errutil.Handle(ctx, err)
		return s.snapshotLocked(), nil
	}
	return nil, err
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
