package slack

// BotUserID returns the bot user ID. It's just for testing.
func (x *Service) BotUserID() string {
	return x.botID
}

// Test type aliases for exported access
type UserIconCache = userIconCache
type UserProfileCache = userProfileCache

// GetIconCache returns the icon cache for testing.
func (x *Service) GetIconCache() map[string]*userIconCache {
	return x.iconCache
}

// SetIconCache sets the icon cache for testing.
func (x *Service) SetIconCache(cache map[string]*userIconCache) {
	x.iconCache = cache
}

// GetProfileCache returns the profile cache for testing.
func (x *Service) GetProfileCache() map[string]*userProfileCache {
	return x.profileCache
}

// SetProfileCache sets the profile cache for testing.
func (x *Service) SetProfileCache(cache map[string]*userProfileCache) {
	x.profileCache = cache
}

// NormalizeChannel exports the private normalizeChannel method for testing.
func (x *Service) NormalizeChannel(channel string) string {
	return x.normalizeChannel(channel)
}
