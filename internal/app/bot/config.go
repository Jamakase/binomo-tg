package bot

import "errors"

// Config holds details necessary for logging.
type Config struct {
	Token string
	Channel string
}

// Validate validates the configuration.
func (c Config) Validate() error {
	if c.Token == "" {
		return errors.New("bot token is required")
	}

	return nil
}
