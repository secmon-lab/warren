package policy

// WaitRefresh blocks until all background refreshes triggered by Snapshot have
// completed. It exists only for deterministic testing of the
// stale-while-revalidate behaviour and is not part of the public API.
func (s *GitHubSource) WaitRefresh() {
	s.refreshWG.Wait()
}
