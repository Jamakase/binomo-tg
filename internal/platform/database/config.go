package database

// Config holds details necessary for logging.
type Config struct {
	// Format specifies the output log format.
	// Accepted values are: json, logfmt
	Port     int64
	Host     string
	Username string
	Password string
}

func (c Config) Validate() error {
	return nil
}
