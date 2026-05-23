package policy

import (
	"context"
	"strings"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

var _ interfaces.PolicyClient = (*Loader)(nil)

// Loader composes one or more Sources into a single PolicyClient. It rebuilds
// the underlying *opaq.Client when the combined version of all sources
// changes, otherwise it returns a cached client.
//
// Loader satisfies interfaces.PolicyClient.
type Loader struct {
	sources []Source

	mu            sync.Mutex
	cachedClient  *opaq.Client
	cachedVersion string
}

// NewLoader creates a Loader from the given sources. At least one source
// SHOULD be supplied, otherwise Query will return ErrNoPolicy.
func NewLoader(sources ...Source) *Loader {
	return &Loader{sources: sources}
}

// Query evaluates the given query string against the merged policy of all
// sources.
func (l *Loader) Query(ctx context.Context, query string, input, output any, opts ...opaq.QueryOption) error {
	client, err := l.client(ctx)
	if err != nil {
		return err
	}
	return client.Query(ctx, query, input, output, opts...)
}

// Sources returns the merged file map (path -> rego content) backing the
// current compiled client. Returns an empty map if no client has been built.
func (l *Loader) Sources() map[string]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cachedClient == nil {
		return map[string]string{}
	}
	return l.cachedClient.Sources()
}

// HasSources reports whether the Loader has any sources configured.
func (l *Loader) HasSources() bool {
	return len(l.sources) > 0
}

// Prime forces an initial build of the underlying opaq client. Calling Prime
// at configure time ensures that Sources() returns the loaded policy contents
// before the first Query, and surfaces any source-side errors (e.g. an
// unreachable GitHub repository) at startup rather than on first evaluation.
func (l *Loader) Prime(ctx context.Context) error {
	_, err := l.client(ctx)
	return err
}

func (l *Loader) client(ctx context.Context) (*opaq.Client, error) {
	if len(l.sources) == 0 {
		return nil, goerr.New("no policy sources configured")
	}

	snapshots := make([]*Snapshot, 0, len(l.sources))
	for _, s := range l.sources {
		snap, err := s.Snapshot(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to obtain policy snapshot")
		}
		snapshots = append(snapshots, snap)
	}

	combined := combineVersions(snapshots)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.cachedClient != nil && l.cachedVersion == combined {
		return l.cachedClient, nil
	}

	merged, err := mergeFiles(snapshots)
	if err != nil {
		return nil, err
	}

	client, err := opaq.New(opaq.DataMap(merged))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to compile merged policy")
	}

	l.cachedClient = client
	l.cachedVersion = combined
	return client, nil
}

func combineVersions(snapshots []*Snapshot) string {
	var b strings.Builder
	for i, s := range snapshots {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(s.Version)
	}
	return b.String()
}

func mergeFiles(snapshots []*Snapshot) (map[string]string, error) {
	out := map[string]string{}
	for _, s := range snapshots {
		for k, v := range s.Files {
			if existing, ok := out[k]; ok && existing != v {
				return nil, goerr.New("duplicate policy file key across sources",
					goerr.V("key", k))
			}
			out[k] = v
		}
	}
	return out, nil
}
