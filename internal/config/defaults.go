package config

// DefaultConfig returns the built-in default configuration.
func DefaultConfig() *Config {
	return &Config{
		Version:      1,
		LogLevel:     "info",
		AuditEnabled: true,
		Patterns: PatternConfig{
			Disabled: []string{},
			Custom:   []CustomPattern{},
		},
		Whitelist: []string{
			"127.0.0.1",
			"0.0.0.0",
			"localhost",
			"example.com",
			"test@example.com",
		},
	}
}
