package config

// ConfigBad has a hardcoded Slack bot token.
type ConfigBad struct {
	SlackBotToken string
}

func LoadBad() *ConfigBad {
	return &ConfigBad{
		// ruleid: slotly-004
		SlackBotToken: "xoxb-1234567890-abcdefghijklmnop",
	}
}
