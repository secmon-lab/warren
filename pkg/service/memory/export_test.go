package memory

// Export internal functions for testing
var (
	CalculateCosineSimilarity = calculateCosineSimilarity
	CalculateRecencyScore     = calculateRecencyScore
	CalculateDaysSinceUsed    = calculateDaysSinceUsed
)
