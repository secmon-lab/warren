package knowledge

import (
	"math"
	"strings"
	"unicode"
)

// bm25Scorer computes BM25 scores for documents against a query.
// Documents are expected to be pre-tokenized.
type bm25Scorer struct {
	k1  float64 // term frequency saturation parameter (typically 1.2-2.0)
	b   float64 // length normalization parameter (typically 0.75)
	idf map[string]float64

	avgDL float64 // average document length
}

type bm25Document struct {
	tokens []string
	length int
}

// newBM25Scorer creates a scorer from a corpus of documents.
// IDF values are precomputed from the corpus.
func newBM25Scorer(docs []bm25Document) *bm25Scorer {
	s := &bm25Scorer{
		k1:  1.5,
		b:   0.75,
		idf: make(map[string]float64),
	}

	n := float64(len(docs))
	if n == 0 {
		return s
	}

	// Calculate average document length
	totalLen := 0
	for _, doc := range docs {
		totalLen += doc.length
	}
	s.avgDL = float64(totalLen) / n

	// Calculate document frequency for each term
	df := make(map[string]int)
	for _, doc := range docs {
		seen := make(map[string]bool)
		for _, token := range doc.tokens {
			if !seen[token] {
				df[token]++
				seen[token] = true
			}
		}
	}

	// Calculate IDF: log((N - df + 0.5) / (df + 0.5) + 1)
	for term, freq := range df {
		s.idf[term] = math.Log((n-float64(freq)+0.5)/(float64(freq)+0.5) + 1.0)
	}

	return s
}

// score computes the BM25 score for a single document against query tokens.
func (s *bm25Scorer) score(doc bm25Document, queryTokens []string) float64 {
	if s.avgDL == 0 {
		return 0
	}

	// Count term frequencies in document
	tf := make(map[string]int)
	for _, token := range doc.tokens {
		tf[token]++
	}

	total := 0.0
	dl := float64(doc.length)

	for _, qToken := range queryTokens {
		freq := float64(tf[qToken])
		if freq == 0 {
			continue
		}

		idf := s.idf[qToken]
		tfNorm := (freq * (s.k1 + 1)) / (freq + s.k1*(1-s.b+s.b*dl/s.avgDL))
		total += idf * tfNorm
	}

	return total
}

// tokenize splits text into lowercase tokens, stripping punctuation.
func tokenize(text string) []string {
	lower := strings.ToLower(text)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '-' && r != '_'
	})
	return words
}
