// FinalScore calculation for Agent Memory
// This mirrors the selection algorithm from pkg/service/memory/selection.go

export interface ScoreBreakdown {
  similarity: number; // 0.0 ~ 1.0
  quality: number; // 0.0 ~ 1.0 (normalized from -10 ~ +10)
  recency: number; // 0.0 ~ 1.0
  finalScore: number; // weighted sum
}

// Algorithm constants matching the backend implementation
const SIMILARITY_WEIGHT = 0.5;
const QUALITY_WEIGHT = 0.3;
const RECENCY_WEIGHT = 0.2;
const RECENCY_HALF_LIFE_DAYS = 30.0;
const SCORE_MIN = -10.0;
const SCORE_MAX = 10.0;

/**
 * Calculate cosine similarity between two vectors
 * Returns a value between 0.0 (orthogonal/unrelated) and 1.0 (identical direction)
 */
export function calculateCosineSimilarity(
  v1: number[],
  v2: number[]
): number {
  if (v1.length !== v2.length || v1.length === 0) {
    return 0.0;
  }

  let dotProduct = 0.0;
  let norm1 = 0.0;
  let norm2 = 0.0;

  for (let i = 0; i < v1.length; i++) {
    dotProduct += v1[i] * v2[i];
    norm1 += v1[i] * v1[i];
    norm2 += v2[i] * v2[i];
  }

  if (norm1 === 0 || norm2 === 0) {
    return 0.0;
  }

  return dotProduct / (Math.sqrt(norm1) * Math.sqrt(norm2));
}

/**
 * Calculate recency score based on exponential decay
 * Returns:
 * - 0.0 if never used (lastUsedAt is null/undefined)
 * - 1.0 if used very recently
 * - 0.5 after halfLifeDays
 * - Exponential decay over time: 0.5^(daysSinceUsed / halfLifeDays)
 */
export function calculateRecencyScore(
  lastUsedAt: string | null | undefined,
  now: Date = new Date()
): number {
  if (!lastUsedAt) {
    return 0.0;
  }

  const lastUsed = new Date(lastUsedAt);
  const daysSince = (now.getTime() - lastUsed.getTime()) / (1000 * 60 * 60 * 24);

  return Math.pow(0.5, daysSince / RECENCY_HALF_LIFE_DAYS);
}

/**
 * Calculate final score for an agent memory
 * @param score - The quality score from the memory (-10 to +10)
 * @param lastUsedAt - When the memory was last used (ISO string or null)
 * @param queryEmbedding - Optional query embedding for similarity calculation
 * @param memoryEmbedding - Optional memory embedding for similarity calculation
 * @param now - Current time (defaults to new Date())
 */
export function calculateFinalScore(
  score: number,
  lastUsedAt: string | null | undefined,
  queryEmbedding?: number[],
  memoryEmbedding?: number[],
  now: Date = new Date()
): ScoreBreakdown {
  // Calculate similarity score
  let similarity = 0.0;
  if (queryEmbedding && memoryEmbedding) {
    similarity = calculateCosineSimilarity(queryEmbedding, memoryEmbedding);
  }

  // Normalize quality score from -10~+10 to 0~1
  const quality = (score - SCORE_MIN) / (SCORE_MAX - SCORE_MIN);

  // Calculate recency score
  const recency = calculateRecencyScore(lastUsedAt, now);

  // Calculate final weighted score
  const finalScore =
    SIMILARITY_WEIGHT * similarity +
    QUALITY_WEIGHT * quality +
    RECENCY_WEIGHT * recency;

  return {
    similarity,
    quality,
    recency,
    finalScore,
  };
}
