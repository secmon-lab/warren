package knowledge

import (
	"testing"

	"github.com/m-mizutani/gt"
)

func TestTokenize(t *testing.T) {
	t.Run("basic tokenization", func(t *testing.T) {
		tokens := tokenize("Hello World")
		gt.A(t, tokens).Length(2)
		gt.V(t, tokens[0]).Equal("hello")
		gt.V(t, tokens[1]).Equal("world")
	})

	t.Run("preserves dots and hyphens", func(t *testing.T) {
		tokens := tokenize("svchost.exe CVE-2024-1234")
		gt.A(t, tokens).Length(2)
		gt.V(t, tokens[0]).Equal("svchost.exe")
		gt.V(t, tokens[1]).Equal("cve-2024-1234")
	})

	t.Run("empty string", func(t *testing.T) {
		tokens := tokenize("")
		gt.A(t, tokens).Length(0)
	})
}

func TestBM25Scorer(t *testing.T) {
	docs := []bm25Document{
		{tokens: tokenize("svchost.exe windows service host process"), length: 5},
		{tokens: tokenize("crowdstrike falcon edr detection"), length: 4},
		{tokens: tokenize("svchost.exe dll injection false positive crowdstrike"), length: 6},
	}

	scorer := newBM25Scorer(docs)

	t.Run("exact match scores higher", func(t *testing.T) {
		query := tokenize("svchost.exe")
		score0 := scorer.score(docs[0], query) // contains svchost.exe
		score1 := scorer.score(docs[1], query) // no match
		gt.True(t, score0 > score1)
		gt.V(t, score1).Equal(0.0)
	})

	t.Run("multi-term query", func(t *testing.T) {
		query := tokenize("svchost.exe crowdstrike")
		score2 := scorer.score(docs[2], query) // has both terms
		score0 := scorer.score(docs[0], query) // only svchost.exe
		score1 := scorer.score(docs[1], query) // only crowdstrike
		gt.True(t, score2 > score0)
		gt.True(t, score2 > score1)
	})

	t.Run("empty corpus", func(t *testing.T) {
		emptyScorer := newBM25Scorer(nil)
		score := emptyScorer.score(bm25Document{tokens: []string{"test"}, length: 1}, []string{"test"})
		gt.V(t, score).Equal(0.0)
	})
}
