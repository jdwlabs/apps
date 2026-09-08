package main

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authhttp"
	"libs/backend/shared/auth/authz"
	"libs/backend/shared/util"
)

// The operational endpoints, outside the authenticated surface as
// SecurityConfig's permitAll matchers put them. /actuator/health and
// /actuator/prometheus keep the paths the JVM serves so probes and the scrape
// configuration carry over unchanged; /health is the path the sibling Go
// services expose and costs nothing to keep.
const (
	healthPath          = "/health"
	actuatorHealthPath  = "/actuator/health"
	actuatorMetricsPath = "/actuator/prometheus"
	actuatorPathPrefix  = "/actuator/"
	authPathPrefix      = "/auth/"
)

var (
	ErrNoStore  = errors.New("the server needs a store")
	ErrNoMinter = errors.New("the server needs a token minter")
)

// ServerConfig is everything the HTTP surface needs. Nothing here is read from
// the environment: main resolves the environment once and hands the result over,
// so a test builds the same server without one.
type ServerConfig struct {
	Store    Store
	Verifier *auth.Verifier
	Minter   *minter
	CORS     CORS
	// Metrics is optional. A server built without one registers its own, so a
	// test never has to and two servers in a process cannot collide.
	Metrics *Metrics
}

type Server struct {
	operations []Operation
	metrics    *Metrics
	handler    http.Handler
}

// NewServer wires the layers in the order the JVM filter chain applies them:
// CORS outermost, then logging and metrics, then authentication, then the
// router that resolves an operation.
//
// The CORS layer has to sit outside authentication. A browser puts no
// Authorization header on a preflight, so a preflight that reached the
// authentication layer would be refused, and every cross-origin call from the
// frontends would fail at cutover with the request itself perfectly valid.
func NewServer(config ServerConfig) (*Server, error) {
	if config.Store == nil {
		return nil, ErrNoStore
	}
	if config.Minter == nil {
		return nil, ErrNoMinter
	}
	metrics := config.Metrics
	if metrics == nil {
		metrics = NewMetrics()
	}

	api := &handlers{
		store:  config.Store,
		minter: config.Minter,
		// ProfileIDForUser is deliberately unset: no operation in this contract
		// authorizes on a profile id, so there is no fallback to resolve and no
		// reason for this service to read auth.profiles on an authorization path.
		authorizer: authz.Authorizer{},
	}
	operations := api.operations()

	routes := make([]Route, 0, len(operations)+3)
	for _, operation := range operations {
		routes = append(routes, Route{
			Method:   operation.Method,
			Pattern:  operation.Pattern,
			Produces: operation.Produces,
			Handler:  operation.Handler,
		})
	}
	routes = append(routes,
		Route{Method: http.MethodGet, Pattern: healthPath, Handler: http.HandlerFunc(health)},
		Route{Method: http.MethodGet, Pattern: actuatorHealthPath, Handler: http.HandlerFunc(health)},
		Route{Method: http.MethodGet, Pattern: actuatorMetricsPath, Handler: metrics.Handler()},
	)

	router, err := NewRouter(routes)
	if err != nil {
		return nil, err
	}

	middleware, err := authhttp.NewMiddleware(config.Verifier)
	if err != nil {
		return nil, err
	}
	middleware.Public = isPublicPath
	middleware.OnError = func(r *http.Request, err error) {
		slog.Warn("a presented token did not verify", "error", err, "method", r.Method, "path", r.URL.Path)
	}
	// Preflight is deliberately left nil: the CORS layer below is the outer one
	// and answers preflights before this middleware sees them.

	handler := config.CORS.Handler(
		util.Logging(
			metrics.Middleware(router,
				middleware.Handler(router))))

	return &Server{operations: operations, metrics: metrics, handler: handler}, nil
}

// Operations lists the served API operations with the rule each is authorized
// under, so the contract suite can compare them against the frozen document.
func (s *Server) Operations() []Operation { return s.operations }

func (s *Server) Handler() http.Handler { return s.handler }

// isPublicPath stands in for SecurityConfig's permitAll matchers. The JVM
// permits /auth/**, /actuator/** and /openapi/**; this service serves no
// /openapi surface, so the first two are the whole list.
//
// A public request whose token does verify still carries its principal, which
// is what the shared middleware does: the JVM's filter runs ahead of the
// matchers too.
func isPublicPath(r *http.Request) bool {
	return r.URL.Path == healthPath ||
		strings.HasPrefix(r.URL.Path, actuatorPathPrefix) ||
		strings.HasPrefix(r.URL.Path, authPathPrefix)
}

// health answers in the shape the actuator health endpoint answers, so a chart
// probe written against the JVM keeps working.
func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "UP"})
}
