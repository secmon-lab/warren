package knowledge

import (
	"math"
	"sort"

	"cloud.google.com/go/firestore"
	knowledgeModel "github.com/secmon-lab/warren/pkg/domain/model/knowledge"
)

// searchResult holds a knowledge with its hybrid search score.
type searchResult struct {
	Knowledge *knowledgeModel.Knowledge
	Score     float64
}

// hybridSearch performs hybrid search (cosine similarity + BM25) with RRF fusion
// on the given documents against a query.
func hybridSearch(docs []*knowledgeModel.Knowledge, queryEmbedding firestore.Vector32, queryText string, limit int) []*knowledgeModel.Knowledge {
	if len(docs) == 0 {
		return nil
	}

	queryTokens := tokenize(queryText)

	// Build BM25 corpus with title boost (2x)
	bm25Docs := make([]bm25Document, len(docs))
	for i, doc := range docs {
		titleTokens := tokenize(doc.Title)
		claimTokens := tokenize(doc.Claim)
		// Title gets 2x boost by duplicating its tokens
		allTokens := make([]string, 0, len(titleTokens)*2+len(claimTokens))
		allTokens = append(allTokens, titleTokens...)
		allTokens = append(allTokens, titleTokens...) // 2x boost
		allTokens = append(allTokens, claimTokens...)
		bm25Docs[i] = bm25Document{
			tokens: allTokens,
			length: len(allTokens),
		}
	}

	scorer := newBM25Scorer(bm25Docs)

	// Rank by cosine similarity
	type ranked struct {
		index int
		score float64
	}

	cosineRanks := make([]ranked, len(docs))
	for i, doc := range docs {
		cosineRanks[i] = ranked{index: i, score: cosineSimilarity(queryEmbedding, doc.Embedding)}
	}
	sort.Slice(cosineRanks, func(i, j int) bool {
		return cosineRanks[i].score > cosineRanks[j].score
	})

	// Rank by BM25
	bm25Ranks := make([]ranked, len(docs))
	for i := range docs {
		bm25Ranks[i] = ranked{index: i, score: scorer.score(bm25Docs[i], queryTokens)}
	}
	sort.Slice(bm25Ranks, func(i, j int) bool {
		return bm25Ranks[i].score > bm25Ranks[j].score
	})

	// RRF fusion (k=60)
	const rrfK = 60.0
	rrfScores := make(map[int]float64, len(docs))

	for rank, r := range cosineRanks {
		rrfScores[r.index] += 1.0 / (rrfK + float64(rank+1))
	}
	for rank, r := range bm25Ranks {
		rrfScores[r.index] += 1.0 / (rrfK + float64(rank+1))
	}

	// Sort by RRF score
	results := make([]searchResult, 0, len(docs))
	for idx, score := range rrfScores {
		results = append(results, searchResult{
			Knowledge: docs[idx],
			Score:     score,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top-K
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	out := make([]*knowledgeModel.Knowledge, len(results))
	for i, r := range results {
		out[i] = r.Knowledge
	}
	return out
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b firestore.Vector32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dotProduct / denom
}
