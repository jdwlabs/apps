package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"libs/backend/shared/auth"
	"libs/backend/shared/servicehttp"
	"libs/backend/shared/util"
)

// The variables usersrole reads, kept byte for byte so the chart work
// downstream reuses the secret and datasource block it already has rather than
// inventing a second set for the same database and the same signing key.
const (
	envDatasourceURL = "UR_PG_DATASOURCE_URL"
	envUsername      = "UR_PG_USERNAME"
	envPassword      = "UR_PG_PASSWORD"
	envTokenTTLMs    = "UR_JWT_EXPIRATION_TIME_MS"
)

// The variables this service adds. They carry its own prefix because they
// describe this process rather than the datasource both share.
const (
	envPort                  = "ID_PORT"
	envIssuerOrigin          = "ID_JWT_ISSUER_ORIGIN"
	envAllowAnyIssuer        = "ID_JWT_ALLOW_ANY_ISSUER_AND_AUDIENCE"
	envMaxConnections        = "ID_DB_MAX_CONNECTIONS"
	envMinConnections        = "ID_DB_MIN_CONNECTIONS"
	envCORSOriginPatterns    = "ID_CORS_ALLOWED_ORIGIN_PATTERNS"
	envCORSAllowedMethods    = "ID_CORS_ALLOWED_METHODS"
	envCORSAllowedHeaders    = "ID_CORS_ALLOWED_HEADERS"
	envShutdownTimeoutSecond = "ID_SHUTDOWN_TIMEOUT_SECONDS"
)

const (
	defaultPort = "8080"
	// HikariCP's maximum of 10 sizes a pool for the whole JVM surface — users,
	// roles and profiles — where a thread holds a connection for the length of a
	// request. This service serves two thirds of that surface and does not bind a
	// connection to a goroutine, so it starts lower and is raised from
	// measurement rather than from the JVM's number.
	defaultMaxConnections    = int32(5)
	defaultMinConnections    = int32(2)
	defaultShutdownTimeoutS  = 10
	defaultTokenTTLMs        = 7200000
	authenticatePathSuffix   = "/auth/authenticate"
	defaultCORSOriginPattern = "http://*:[*],https://*:[*]"
	defaultCORSMethods       = "GET,POST,PUT,DELETE,HEAD,PATCH,OPTIONS"
	defaultCORSHeaders       = "Authorization,Content-Type"
)

var (
	ErrUnsupportedDatasource = errors.New("the datasource url is not a postgresql url")
	ErrNoDatasource          = errors.New(envDatasourceURL + " is empty or unset")
	ErrNoIssuerOrigin        = errors.New(envIssuerOrigin + " is empty; it is the origin every token is stamped with")
)

// Config is the whole environment this process reads, resolved once at startup
// so that nothing below main has to reach for an environment variable.
type Config struct {
	Address        string
	DatabaseDSN    string
	MaxConnections int32
	MinConnections int32

	SecretKeyBase64 string
	// IssuerOrigin is what this service stamps into every token it mints, and
	// what the two expected values below are derived from.
	IssuerOrigin              string
	ExpectedIssuer            string
	ExpectedAudience          string
	AllowAnyIssuerAndAudience bool
	TokenTTL                  time.Duration

	CORS                   servicehttp.CORS
	ShutdownTimeoutSeconds int
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
		MaxConnections:  envInt32(envMaxConnections, defaultMaxConnections),
		MinConnections:  envInt32(envMinConnections, defaultMinConnections),
		SecretKeyBase64: secret,
		TokenTTL:        time.Duration(envInt(envTokenTTLMs, defaultTokenTTLMs)) * time.Millisecond,
		CORS: servicehttp.CORS{
			AllowedOriginPatterns: envList(envCORSOriginPatterns, defaultCORSOriginPattern),
			AllowedMethods:        envList(envCORSAllowedMethods, defaultCORSMethods),
			AllowedHeaders:        envList(envCORSAllowedHeaders, defaultCORSHeaders),
		},
		ShutdownTimeoutSeconds: envInt(envShutdownTimeoutSecond, defaultShutdownTimeoutS),
	}

	// The issuer origin is required whatever the verification settings say,
	// because this service is the one that mints. The JVM derives it from the
	// request instead — scheme, server name and port off the incoming Host — and
	// gets away with it because it never checks iss on the way back in. Here the
	// claim is verified, so deriving it from a caller-controlled header would
	// let one caller mint a token the next request refuses. Configured once,
	// the mint and the check cannot disagree.
	origin := strings.TrimSuffix(util.GetEnvOrDefault(envIssuerOrigin, ""), "/")
	if origin == "" {
		return Config{}, ErrNoIssuerOrigin
	}
	config.IssuerOrigin = origin

	config.AllowAnyIssuerAndAudience = util.GetEnvOrDefault(envAllowAnyIssuer, "") == "true"
	if !config.AllowAnyIssuerAndAudience {
		config.ExpectedIssuer = origin + authenticatePathSuffix
		config.ExpectedAudience = origin
	}
	// Left empty otherwise: the verifier refuses the flag alongside an expected
	// value, and the flag relaxes verification only — nothing about minting.

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

// envInt32 parses at the width the pool configuration takes. Parsing as int and
// converting would truncate a value above 2^31 into a plausible small pool on a
// 64-bit host rather than falling back, so the bound is the parser's rather than
// a check after the fact.
func envInt32(name string, fallback int32) int32 {
	value, err := strconv.ParseInt(util.GetEnvOrDefault(name, ""), 10, 32)
	if err != nil || value <= 0 {
		return fallback
	}
	return int32(value)
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
