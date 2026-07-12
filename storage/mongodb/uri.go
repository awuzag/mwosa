package mongodb

import (
	"net/url"
	"strings"

	"github.com/samber/oops"
)

// DatabaseNameFromURI returns the database path component from a MongoDB URI.
// URIs without a database path return an empty string.
func DatabaseNameFromURI(uri string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(uri))
	if err != nil {
		return "", oops.In("mongodb_uri").Wrapf(err, "parse MongoDB URI")
	}

	path := strings.TrimPrefix(parsed.EscapedPath(), "/")
	if path == "" {
		return "", nil
	}
	if strings.Contains(path, "/") {
		return "", oops.In("mongodb_uri").New("MongoDB URI database path must contain a single segment")
	}

	database, err := url.PathUnescape(path)
	if err != nil {
		return "", oops.In("mongodb_uri").Wrapf(err, "decode MongoDB URI database")
	}
	if strings.Contains(database, "/") {
		return "", oops.In("mongodb_uri").New("MongoDB URI database name must not contain slash")
	}

	return strings.TrimSpace(database), nil
}

// URIWithDatabase returns uri with its database path replaced by database.
func URIWithDatabase(uri string, database string) (string, error) {
	database = strings.TrimSpace(database)
	if database == "" {
		return "", oops.In("mongodb_uri").New("MongoDB database is required")
	}

	parsed, err := url.Parse(strings.TrimSpace(uri))
	if err != nil {
		return "", oops.In("mongodb_uri").Wrapf(err, "parse MongoDB URI")
	}
	parsed.Path = "/" + url.PathEscape(database)

	return parsed.String(), nil
}

// DevelopmentURI returns uri with the database name scoped by hostname.
func DevelopmentURI(uri string, hostname string) (string, error) {
	database, err := DatabaseNameFromURI(uri)
	if err != nil {
		return "", err
	}
	if database == "" {
		database = DefaultDatabaseName
	}

	scoped, err := DevelopmentDatabaseName(database, hostname)
	if err != nil {
		return "", err
	}

	return URIWithDatabase(uri, scoped)
}

// DevelopmentDatabaseName prefixes database with a sanitized hostname.
func DevelopmentDatabaseName(database string, hostname string) (string, error) {
	database = strings.TrimSpace(database)
	if database == "" {
		return "", oops.In("mongodb_config").New("MongoDB database is required")
	}

	prefix := sanitizeDatabasePrefix(hostname)
	if prefix == "" {
		return "", oops.In("mongodb_config").New("development hostname is required")
	}

	if strings.HasPrefix(database, prefix+"-") {
		return database, nil
	}

	return prefix + "-" + database, nil
}

func sanitizeDatabasePrefix(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		valid := char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_'
		if valid {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-_")
}
