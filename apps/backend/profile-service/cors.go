package main

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

// CORS reproduces the CorsFilter that SecurityConfig installs ahead of the JWT
// filter, with the origin patterns, methods and headers its
// CorsConfigurationSource registers for "/**".
//
// The shared authentication middleware deliberately emits no CORS headers, so
// without this layer every cross-origin call from the frontends fails at
// cutover even though the request itself would have been authorized.
type CORS struct {
	// AllowedOriginPatterns uses Spring's syntax, where * matches within the
	// origin and a :[...] suffix constrains the port.
	AllowedOriginPatterns []string
	AllowedMethods        []string
	AllowedHeaders        []string
}

const invalidCORSRequest = "Invalid CORS request"

// Handler wraps next. It must be the outermost layer of the two, matching the
// filter order in the JVM: a preflight carries no Authorization header, so
// authenticating it would refuse it before this layer could answer it.
func (c CORS) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Added to every response, as DefaultCorsProcessor adds them, so a
		// shared cache cannot serve one origin's response to another.
		for _, header := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
			w.Header().Add("Vary", header)
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		preflight := isPreflight(r)
		if !c.allowsOrigin(origin) {
			rejectCORS(w)
			return
		}
		method := r.Method
		if preflight {
			method = r.Header.Get("Access-Control-Request-Method")
		}
		if !c.allowsMethod(method) {
			rejectCORS(w)
			return
		}

		if preflight {
			// Only a preflight is refused for its headers. An actual request
			// carries headers the browser has already cleared, and Spring does
			// not re-check them.
			allowed, ok := c.allowedRequestHeaders(r.Header.Get("Access-Control-Request-Headers"))
			if !ok {
				rejectCORS(w)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.AllowedMethods, ", "))
			if allowed != "" {
				w.Header().Set("Access-Control-Allow-Headers", allowed)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		next.ServeHTTP(w, r)
	})
}

// isPreflight matches CorsUtils.isPreFlightRequest, so this layer and the
// shared middleware agree on what a preflight is.
func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Origin") != "" &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

func rejectCORS(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	if _, err := w.Write([]byte(invalidCORSRequest)); err != nil {
		slog.Error("could not write the CORS refusal", "error", err)
	}
}

func (c CORS) allowsOrigin(origin string) bool {
	for _, pattern := range c.AllowedOriginPatterns {
		if matchesOriginPattern(pattern, origin) {
			return true
		}
	}
	return false
}

func (c CORS) allowsMethod(method string) bool {
	for _, allowed := range c.AllowedMethods {
		if strings.EqualFold(allowed, method) {
			return true
		}
	}
	return false
}

// allowedRequestHeaders echoes back the requested headers this configuration
// permits, as DefaultCorsProcessor does, and reports false when one is not
// permitted. Header names are compared case-insensitively because a browser
// lower-cases them on the preflight.
func (c CORS) allowedRequestHeaders(requested string) (string, bool) {
	if strings.TrimSpace(requested) == "" {
		return "", true
	}
	echoed := make([]string, 0, len(c.AllowedHeaders))
	for _, entry := range strings.Split(requested, ",") {
		name := strings.TrimSpace(entry)
		if name == "" {
			continue
		}
		permitted := false
		for _, allowed := range c.AllowedHeaders {
			if strings.EqualFold(allowed, name) {
				permitted = true
				break
			}
		}
		if !permitted {
			return "", false
		}
		echoed = append(echoed, name)
	}
	return strings.Join(echoed, ", "), true
}

// matchesOriginPattern decides one of Spring's allowed-origin patterns. The
// syntax is not a glob: `*` matches any run of characters within the origin,
// and a trailing `:[...]` names the ports, with `[*]` standing for any port or
// none at all.
func matchesOriginPattern(pattern, origin string) bool {
	host, ports := pattern, ""
	if start := strings.Index(pattern, ":["); start != -1 && strings.HasSuffix(pattern, "]") {
		host = pattern[:start]
		ports = pattern[start+2 : len(pattern)-1]
	}

	expression := "^"
	for i, literal := range strings.Split(host, "*") {
		if i > 0 {
			expression += ".*"
		}
		expression += regexp.QuoteMeta(literal)
	}
	switch {
	case ports == "*":
		expression += `(:\d+)?`
	case ports != "":
		quoted := make([]string, 0, 4)
		for _, port := range strings.Split(ports, ",") {
			quoted = append(quoted, regexp.QuoteMeta(strings.TrimSpace(port)))
		}
		expression += ":(" + strings.Join(quoted, "|") + ")"
	}
	expression += "$"

	matched, err := regexp.MatchString(expression, origin)
	if err != nil {
		// Unreachable: every metacharacter above is either quoted or written
		// here, so the expression always compiles. Refusing on the impossible
		// keeps a future edit from failing open.
		slog.Error("an origin pattern did not compile", "pattern", pattern, "error", err)
		return false
	}
	return matched
}
