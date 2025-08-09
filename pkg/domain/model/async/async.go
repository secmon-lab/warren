package async

// Config represents configuration for asynchronous alert hooks
type Config struct {
	Raw    bool
	PubSub bool
	SNS    bool
}
