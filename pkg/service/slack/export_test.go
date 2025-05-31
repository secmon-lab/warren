package slack

// BotUserID returns the bot user ID. It's just for testing.
func (x *Service) BotUserID() string {
	return x.botID
}

// GetIconCache returns the icon cache for testing.
func (x *Service) GetIconCache() map[string]*UserIconCache {
	return x.iconCache
}

// SetIconCache sets the icon cache for testing.
func (x *Service) SetIconCache(cache map[string]*UserIconCache) {
	x.iconCache = cache
}
