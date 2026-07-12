package mongodb

import (
	"os"
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
	// Development scopes the database name to the local host so multiple
	// development checkouts can share one MongoDB server without sharing data.
	Development bool
	Hostname    string
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
	c.Hostname = strings.TrimSpace(c.Hostname)
	if c.Database == "" {
		database, err := DatabaseNameFromURI(c.URI)
		if err != nil {
			return Config{}, err
		}
		c.Database = database
	}
	if c.Database == "" {
		c.Database = DefaultDatabaseName
	}
	if c.Development {
		hostname := c.Hostname
		if hostname == "" {
			resolved, err := os.Hostname()
			if err != nil {
				return Config{}, oops.In("mongodb_config").Wrapf(err, "resolve development hostname")
			}
			hostname = resolved
		}
		database, err := DevelopmentDatabaseName(c.Database, hostname)
		if err != nil {
			return Config{}, err
		}
		c.Database = database
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
