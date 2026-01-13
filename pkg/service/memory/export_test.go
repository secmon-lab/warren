package memory

// Export internal functions for testing
var (
	CalculateCosineSimilarity = calculateCosineSimilarity
	CalculateRecencyScore     = calculateRecencyScore
	CalculateDaysSinceUsed    = calculateDaysSinceUsed
)

// EnableAsyncTrackingForTest enables async tracking for testing (unexposed)
func (s *Service) EnableAsyncTrackingForTest() *Service {
	s.enableAsyncTracking = true
	return s
}

// WaitForAsyncOperationsForTest waits for async operations (unexposed, for testing only)
func (s *Service) WaitForAsyncOperationsForTest() {
	if s.enableAsyncTracking {
		s.asyncWg.Wait()
	}
}
