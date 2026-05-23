package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
)

// FileSource loads Rego files from local filesystem paths once at
// construction time. The contents do not change for the lifetime of the
// process, matching the existing --policy behavior.
type FileSource struct {
	paths    []string
	snapshot *Snapshot
}

// NewFileSource reads .rego files from the given paths (files or directories,
// recursively) using opaq.Files semantics. Returns an error if any path is
// invalid or files fail to load.
func NewFileSource(paths []string) (*FileSource, error) {
	client, err := opaq.New(opaq.Files(paths...))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to load policy files",
			goerr.V("paths", paths))
	}

	files := client.Sources()
	return &FileSource{
		paths: paths,
		snapshot: &Snapshot{
			Files:   files,
			Version: "file:" + hashFiles(files),
		},
	}, nil
}

func (s *FileSource) Snapshot(_ context.Context) (*Snapshot, error) {
	return s.snapshot, nil
}

// hashFiles produces a deterministic version identifier for a files map.
func hashFiles(files map[string]string) string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(files[k]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
