package mongodb

import (
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	DefaultAppName      = "mwosa"
	DefaultDatabaseName = "mwosa"
	DefaultTimeout      = 10 * time.Second
)

type Config struct {
	URI      string
	Database string
	AppName  string
	Timeout  time.Duration
}

func (c Config) Validate() error {
	errb := oops.In("mongodb_config")
	if strings.TrimSpace(c.URI) == "" {
		return errb.New("mongodb URI is required")
	}
	if strings.TrimSpace(c.Database) == "" {
		return errb.New("mongodb database name is required")
	}
	return nil
}

func (c Config) WithDefaults() (Config, error) {
	c.URI = strings.TrimSpace(c.URI)
	c.Database = strings.TrimSpace(c.Database)
	c.AppName = strings.TrimSpace(c.AppName)
	if c.Database == "" {
		c.Database = DefaultDatabaseName
	}
	if c.AppName == "" {
		c.AppName = DefaultAppName
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}
