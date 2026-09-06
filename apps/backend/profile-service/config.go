package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"libs/backend/shared/auth"
	"libs/backend/shared/util"
)

// The variables usersrole reads, kept byte for byte so the chart work
// downstream reuses the secret and datasource block it already has rather than
// inventing a second set for the same database and the same signing key.
const (
	envDatasourceURL = "UR_PG_DATASOURCE_URL"
	envUsername      = "UR_PG_USERNAME"
	envPassword      = "UR_PG_PASSWORD"
)

// The variables this service adds. They carry its own prefix because they
// describe this process rather than the datasource both share.
const (
	envPort                  = "PS_PORT"
	envIssuerOrigin          = "PS_JWT_ISSUER_ORIGIN"
	envAllowAnyIssuer        = "PS_JWT_ALLOW_ANY_ISSUER_AND_AUDIENCE"
	envMaxConnections        = "PS_DB_MAX_CONNECTIONS"
	envMinConnections        = "PS_DB_MIN_CONNECTIONS"
	envCORSOriginPatterns    = "PS_CORS_ALLOWED_ORIGIN_PATTERNS"
	envCORSAllowedMethods    = "PS_CORS_ALLOWED_METHODS"
	envCORSAllowedHeaders    = "PS_CORS_ALLOWED_HEADERS"
	envShutdownTimeoutSecond = "PS_SHUTDOWN_TIMEOUT_SECONDS"
)

const (
	defaultPort = "8080"
	// HikariCP's maximum of 10 sizes a pool for the whole JVM surface — users,
	// roles and profiles — where a thread holds a connection for the length of a
	// request. This service serves the profile third of that surface and does
	// not bind a connection to a goroutine, so it starts lower and is raised
	// from measurement rather than from the JVM's number.
	defaultMaxConnections    = 5
	defaultMinConnections    = 2
	defaultShutdownTimeoutS  = 10
	authenticatePathSuffix   = "/auth/authenticate"
	defaultCORSOriginPattern = "http://*:[*],https://*:[*]"
	defaultCORSMethods       = "GET,POST,PUT,DELETE,HEAD,PATCH,OPTIONS"
	defaultCORSHeaders       = "Authorization,Content-Type"
)

var (
	ErrUnsupportedDatasource = errors.New("the datasource url is not a postgresql url")
	ErrNoDatasource          = errors.New(envDatasourceURL + " is empty or unset")
)

// Config is the whole environment this process reads, resolved once at startup
// so that nothing below main has to reach for an environment variable.
type Config struct {
	Address                   string
	DatabaseDSN               string
	MaxConnections            int32
	MinConnections            int32
	SecretKeyBase64           string
	ExpectedIssuer            string
	ExpectedAudience          string
	AllowAnyIssuerAndAudience bool
	CORS                      CORS
	ShutdownTimeoutSeconds    int
}

func configFromEnvironment() (Config, error) {
	secret, err := auth.SecretKeyFromEnv()
	if err != nil {
		return Config{}, err
	}

	datasourceURL := util.GetEnvOrDefault(envDatasourceURL, "")
	if datasourceURL == "" {
		return Config{}, ErrNoDatasource
	}
	dsn, err := datasourceDSN(datasourceURL,
		util.GetEnvOrDefault(envUsername, ""), util.GetEnvOrDefault(envPassword, ""))
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Address:         ":" + util.GetEnvOrDefault(envPort, defaultPort),
		DatabaseDSN:     dsn,
		MaxConnections:  int32(envInt(envMaxConnections, defaultMaxConnections)),
		MinConnections:  int32(envInt(envMinConnections, defaultMinConnections)),
		SecretKeyBase64: secret,
		CORS: CORS{
			AllowedOriginPatterns: envList(envCORSOriginPatterns, defaultCORSOriginPattern),
			AllowedMethods:        envList(envCORSAllowedMethods, defaultCORSMethods),
			AllowedHeaders:        envList(envCORSAllowedHeaders, defaultCORSHeaders),
		},
		ShutdownTimeoutSeconds: envInt(envShutdownTimeoutSecond, defaultShutdownTimeoutS),
	}

	// The issuer origin is the scheme://host:port identity-service mints from.
	// The JVM stamps it into aud verbatim and into iss with the authenticate
	// path appended, so both expected values come from the one variable.
	origin := util.GetEnvOrDefault(envIssuerOrigin, "")
	config.AllowAnyIssuerAndAudience = util.GetEnvOrDefault(envAllowAnyIssuer, "") == "true"
	switch {
	case config.AllowAnyIssuerAndAudience && origin != "":
		return Config{}, fmt.Errorf("%s is set alongside %s; they state opposite intentions",
			envAllowAnyIssuer, envIssuerOrigin)
	case config.AllowAnyIssuerAndAudience:
		// Left empty deliberately: the verifier refuses the flag alongside an
		// expected value.
	case origin == "":
		return Config{}, fmt.Errorf("%s is empty; set it, or %s=true to accept a token from any issuer",
			envIssuerOrigin, envAllowAnyIssuer)
	default:
		config.ExpectedIssuer = strings.TrimSuffix(origin, "/") + authenticatePathSuffix
		config.ExpectedAudience = strings.TrimSuffix(origin, "/")
	}

	return config, nil
}

// datasourceDSN turns the JDBC URL Spring reads into the libpq URL pgx reads,
// folding in the credentials Spring keeps in two separate variables. Keeping
// the JDBC form on the way in is what lets one chart value feed both services
// through the cutover.
func datasourceDSN(datasourceURL, username, password string) (string, error) {
	trimmed := strings.TrimPrefix(datasourceURL, "jdbc:")
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%s is not a url: %w", envDatasourceURL, err)
	}
	if parsed.Scheme != "postgresql" && parsed.Scheme != "postgres" {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDatasource, parsed.Scheme)
	}
	if username != "" {
		parsed.User = url.UserPassword(username, password)
	}
	return parsed.String(), nil
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(util.GetEnvOrDefault(name, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envList(name, fallback string) []string {
	entries := strings.Split(util.GetEnvOrDefault(name, fallback), ",")
	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		if trimmed := strings.TrimSpace(entry); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
