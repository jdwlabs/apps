package main

import (
	"strings"
	"testing"
)

func TestTheDatasourceUrlIsReadInTheFormSpringReadsIt(t *testing.T) {
	// The chart hands this service the same three variables it hands usersrole,
	// so the value arrives as a JDBC URL with the credentials alongside it
	// rather than embedded.
	cases := []struct {
		name     string
		url      string
		username string
		password string
		want     string
	}{
		{
			name: "a deployed JDBC url", url: "jdbc:postgresql://authdb:5432/jdw",
			username: "jdw", password: "secret",
			want: "postgresql://jdw:secret@authdb:5432/jdw",
		},
		{
			name: "query parameters survive", url: "jdbc:postgresql://authdb:5432/jdw?sslmode=require",
			username: "jdw", password: "secret",
			want: "postgresql://jdw:secret@authdb:5432/jdw?sslmode=require",
		},
		{
			name: "a password needing escaping", url: "jdbc:postgresql://authdb:5432/jdw",
			username: "jdw", password: "p@ss/word",
			want: "postgresql://jdw:p%40ss%2Fword@authdb:5432/jdw",
		},
		{
			name: "a url already in libpq form is left alone", url: "postgresql://authdb:5432/jdw",
			username: "jdw", password: "secret",
			want: "postgresql://jdw:secret@authdb:5432/jdw",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := datasourceDSN(tc.url, tc.username, tc.password)

			if err != nil {
				t.Fatalf("datasourceDSN: %v", err)
			}
			if got != tc.want {
				t.Errorf("dsn = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestADatasourceUrlForAnotherDatabaseIsRefused(t *testing.T) {
	// Failing at startup beats connecting to nothing and reporting healthy.
	if _, err := datasourceDSN("jdbc:mysql://authdb:3306/jdw", "jdw", "secret"); err == nil {
		t.Error("datasourceDSN accepted a URL for another database")
	}
}

func TestTheConfigurationReadsTheVariablesUsersroleReads(t *testing.T) {
	// Sharing the names is what lets the chart work downstream reuse the secret
	// and the datasource block it already has rather than inventing a second set.
	t.Setenv("UR_JWT_SECRET_KEY", paritySecret)
	t.Setenv("UR_PG_DATASOURCE_URL", "jdbc:postgresql://authdb:5432/jdw")
	t.Setenv("UR_PG_USERNAME", "jdw")
	t.Setenv("UR_PG_PASSWORD", "secret")
	t.Setenv("PS_JWT_ISSUER_ORIGIN", "https://auth.example.com")

	config, err := configFromEnvironment()

	if err != nil {
		t.Fatalf("configFromEnvironment: %v", err)
	}
	if config.SecretKeyBase64 != paritySecret {
		t.Error("the JWT secret was not read from UR_JWT_SECRET_KEY")
	}
	if config.DatabaseDSN != "postgresql://jdw:secret@authdb:5432/jdw" {
		t.Errorf("dsn = %q, want the one built from the UR_PG_* variables", config.DatabaseDSN)
	}
	if config.ExpectedIssuer != "https://auth.example.com/auth/authenticate" {
		t.Errorf("issuer = %q, want the origin with the authenticate path", config.ExpectedIssuer)
	}
	if config.ExpectedAudience != "https://auth.example.com" {
		t.Errorf("audience = %q, want the origin", config.ExpectedAudience)
	}
	if config.Address != ":8080" {
		t.Errorf("address = %q, want :8080", config.Address)
	}
	if config.MaxConnections < 1 || config.MaxConnections > 10 {
		t.Errorf("max connections = %d, want a bounded default", config.MaxConnections)
	}
	if len(config.CORS.AllowedOriginPatterns) != 2 {
		t.Errorf("origin patterns = %v, want the two SecurityConfig registers", config.CORS.AllowedOriginPatterns)
	}
	if strings.Join(config.CORS.AllowedMethods, ",") != "GET,POST,PUT,DELETE,HEAD,PATCH,OPTIONS" {
		t.Errorf("methods = %v, want the seven SecurityConfig registers", config.CORS.AllowedMethods)
	}
	if strings.Join(config.CORS.AllowedHeaders, ",") != "Authorization,Content-Type" {
		t.Errorf("headers = %v, want the two SecurityConfig registers", config.CORS.AllowedHeaders)
	}
}

func TestTheConfigurationRefusesToStartWithNoSecret(t *testing.T) {
	t.Setenv("UR_JWT_SECRET_KEY", "")
	t.Setenv("UR_PG_DATASOURCE_URL", "jdbc:postgresql://authdb:5432/jdw")
	t.Setenv("PS_JWT_ISSUER_ORIGIN", "https://auth.example.com")

	if _, err := configFromEnvironment(); err == nil {
		t.Error("configFromEnvironment accepted an empty secret")
	}
}

func TestTheConfigurationRefusesToStartWithNoExpectedIssuer(t *testing.T) {
	// An unset expected issuer is far more often an oversight than a decision,
	// and the failure it causes — accepting a token from anywhere — is silent.
	t.Setenv("UR_JWT_SECRET_KEY", paritySecret)
	t.Setenv("UR_PG_DATASOURCE_URL", "jdbc:postgresql://authdb:5432/jdw")
	t.Setenv("PS_JWT_ISSUER_ORIGIN", "")
	t.Setenv("PS_JWT_ALLOW_ANY_ISSUER_AND_AUDIENCE", "")

	if _, err := configFromEnvironment(); err == nil {
		t.Error("configFromEnvironment accepted a verifier that would trust any issuer")
	}
}

func TestAcceptingAnyIssuerHasToBeSaidOutLoud(t *testing.T) {
	t.Setenv("UR_JWT_SECRET_KEY", paritySecret)
	t.Setenv("UR_PG_DATASOURCE_URL", "jdbc:postgresql://authdb:5432/jdw")
	t.Setenv("PS_JWT_ISSUER_ORIGIN", "")
	t.Setenv("PS_JWT_ALLOW_ANY_ISSUER_AND_AUDIENCE", "true")

	config, err := configFromEnvironment()

	if err != nil {
		t.Fatalf("configFromEnvironment: %v", err)
	}
	if !config.AllowAnyIssuerAndAudience {
		t.Error("the flag was set and the configuration did not carry it")
	}
	if config.ExpectedIssuer != "" || config.ExpectedAudience != "" {
		t.Error("an expected issuer was kept alongside the flag; the verifier refuses that pairing")
	}
}

func TestTheCorsListsCanBeOverriddenForADeploymentThatNarrowsThem(t *testing.T) {
	t.Setenv("UR_JWT_SECRET_KEY", paritySecret)
	t.Setenv("UR_PG_DATASOURCE_URL", "jdbc:postgresql://authdb:5432/jdw")
	t.Setenv("PS_JWT_ISSUER_ORIGIN", "https://auth.example.com")
	t.Setenv("PS_CORS_ALLOWED_ORIGIN_PATTERNS", "https://app.example.com:[443]")
	t.Setenv("PS_CORS_ALLOWED_METHODS", "GET,POST")
	t.Setenv("PS_CORS_ALLOWED_HEADERS", "Authorization")

	config, err := configFromEnvironment()

	if err != nil {
		t.Fatalf("configFromEnvironment: %v", err)
	}
	if strings.Join(config.CORS.AllowedOriginPatterns, ",") != "https://app.example.com:[443]" {
		t.Errorf("origin patterns = %v, want the overridden one", config.CORS.AllowedOriginPatterns)
	}
	if strings.Join(config.CORS.AllowedMethods, ",") != "GET,POST" {
		t.Errorf("methods = %v, want the overridden pair", config.CORS.AllowedMethods)
	}
	if strings.Join(config.CORS.AllowedHeaders, ",") != "Authorization" {
		t.Errorf("headers = %v, want the overridden one", config.CORS.AllowedHeaders)
	}
}
