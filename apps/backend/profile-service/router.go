package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Route is one operation: a method, a path pattern with {name} captures, and
// optionally the single media type the handler produces.
type Route struct {
	Method string
	// Pattern is a rooted path whose captures are written {name}, matching the
	// syntax the frozen contract uses for its paths.
	Pattern string
	// Produces mirrors a Spring @GetMapping(produces = ...) declaration: a
	// request whose Accept header excludes it is answered 406 rather than
	// served. Empty means the operation makes no such promise.
	Produces string
	Handler  http.Handler
}

// Router resolves a request to a Route the way Spring's
// RequestMappingHandlerMapping resolves it, because the two have to agree on
// paths where more than one pattern matches.
//
// net/http's ServeMux cannot express this. It refuses any two patterns where
// neither matches a strict subset of the other, and /api/profiles/by-user/{userId}
// against /api/profiles/{profileId}/icon is exactly that shape — they overlap on
// /api/profiles/by-user/icon and neither contains the other. Registering both
// there panics, so the service could not start at all.
type Router struct {
	routes []compiledRoute
}

type compiledRoute struct {
	Route
	segments []segment
	// captures and normalizedLength are PathPattern.SPECIFICITY_COMPARATOR's two
	// remaining keys once catch-alls are excluded, which they are here: no
	// pattern in either contract uses one.
	captures         int
	normalizedLength int
}

type segment struct {
	literal string
	capture string
}

func (s segment) isCapture() bool { return s.capture != "" }

// NewRouter compiles the routes and refuses any pair that a request could match
// with nothing to separate them.
//
// Spring discovers such a pair at request time, throwing
// IllegalStateException: Ambiguous handler methods mapped and answering 500 to
// whoever asked. Refusing at construction turns that into a failure to start,
// which a deployment notices and a user never sees.
func NewRouter(routes []Route) (*Router, error) {
	compiled := make([]compiledRoute, 0, len(routes))
	for _, route := range routes {
		c, err := compile(route)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, c)
	}
	if err := refuseAmbiguity(compiled); err != nil {
		return nil, err
	}
	return &Router{routes: compiled}, nil
}

// Patterns lists the method and pattern of every registered route, so a test
// can compare the served operation set against the contract's.
func (r *Router) Patterns() []string {
	patterns := make([]string, 0, len(r.routes))
	for _, route := range r.routes {
		patterns = append(patterns, route.Method+" "+route.Pattern)
	}
	sort.Strings(patterns)
	return patterns
}

// Match reports the pattern that would serve this request. The metrics layer
// labels by the pattern rather than the path so that every id does not open its
// own time series.
func (r *Router) Match(request *http.Request) (string, bool) {
	resolved, status := r.resolve(request)
	if status != http.StatusOK {
		return "", false
	}
	return resolved.Pattern, true
}

func (r *Router) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	resolved, status := r.resolve(request)
	if status != http.StatusOK {
		if status == http.StatusMethodNotAllowed {
			w.Header().Set("Allow", strings.Join(r.methodsFor(request.URL.Path), ", "))
		}
		// No body and no Content-Type: the deployed service writes these
		// through sendError, and server.error.include-message is unset.
		w.WriteHeader(status)
		return
	}
	for i, seg := range resolved.segments {
		if seg.isCapture() {
			request.SetPathValue(seg.capture, pathSegments(request.URL.Path)[i])
		}
	}
	resolved.Handler.ServeHTTP(w, request)
}

// resolve reproduces RequestMappingHandlerMapping.handleNoMatch's order: a
// pattern that matches nothing is 404, a pattern matched by a method that is
// not mapped is 405, and a mapping dropped only by its produces condition is
// 406. Getting the order wrong would report a missing route as a wrong method.
func (r *Router) resolve(request *http.Request) (compiledRoute, int) {
	segments := pathSegments(request.URL.Path)

	byPath := make([]compiledRoute, 0, len(r.routes))
	for _, route := range r.routes {
		if route.matches(segments) {
			byPath = append(byPath, route)
		}
	}
	if len(byPath) == 0 {
		return compiledRoute{}, http.StatusNotFound
	}

	byMethod := make([]compiledRoute, 0, len(byPath))
	for _, route := range byPath {
		if route.Method == request.Method {
			byMethod = append(byMethod, route)
		}
	}
	if len(byMethod) == 0 {
		return compiledRoute{}, http.StatusMethodNotAllowed
	}

	acceptable := make([]compiledRoute, 0, len(byMethod))
	for _, route := range byMethod {
		if accepts(request.Header.Get("Accept"), route.Produces) {
			acceptable = append(acceptable, route)
		}
	}
	if len(acceptable) == 0 {
		return compiledRoute{}, http.StatusNotAcceptable
	}

	best := acceptable[0]
	for _, route := range acceptable[1:] {
		if moreSpecific(route, best) {
			best = route
		}
	}
	return best, http.StatusOK
}

func (r *Router) methodsFor(path string) []string {
	segments := pathSegments(path)
	seen := map[string]bool{}
	for _, route := range r.routes {
		if route.matches(segments) {
			seen[route.Method] = true
		}
	}
	methods := make([]string, 0, len(seen))
	for method := range seen {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods
}

func compile(route Route) (compiledRoute, error) {
	if !strings.HasPrefix(route.Pattern, "/") {
		return compiledRoute{}, fmt.Errorf("route %s %s: pattern must be rooted", route.Method, route.Pattern)
	}
	if route.Handler == nil {
		return compiledRoute{}, fmt.Errorf("route %s %s: no handler", route.Method, route.Pattern)
	}
	raw := pathSegments(route.Pattern)
	segments := make([]segment, 0, len(raw))
	captures, normalized := 0, 0
	for _, text := range raw {
		if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
			name := text[1 : len(text)-1]
			if name == "" {
				return compiledRoute{}, fmt.Errorf("route %s %s: an unnamed capture", route.Method, route.Pattern)
			}
			segments = append(segments, segment{capture: name})
			captures++
			// One character, as PathPattern counts a capture when it normalizes
			// a pattern's length.
			normalized += 2
			continue
		}
		segments = append(segments, segment{literal: text})
		normalized += len(text) + 1
	}
	return compiledRoute{Route: route, segments: segments, captures: captures, normalizedLength: normalized}, nil
}

func (c compiledRoute) matches(segments []string) bool {
	if len(segments) != len(c.segments) {
		return false
	}
	for i, seg := range c.segments {
		if !seg.isCapture() && seg.literal != segments[i] {
			return false
		}
	}
	return true
}

// moreSpecific decides the comparison PathPattern.SPECIFICITY_COMPARATOR makes:
// fewer captures wins, and a longer normalized pattern wins among equals.
func moreSpecific(candidate, incumbent compiledRoute) bool {
	if candidate.captures != incumbent.captures {
		return candidate.captures < incumbent.captures
	}
	return candidate.normalizedLength > incumbent.normalizedLength
}

// refuseAmbiguity reports any two same-method routes a single path could match
// where neither is more specific. Their produces conditions are allowed to
// settle it, as Spring lets them, but nothing else is.
func refuseAmbiguity(routes []compiledRoute) error {
	for i, left := range routes {
		for _, right := range routes[i+1:] {
			if left.Method != right.Method || !canCollide(left, right) {
				continue
			}
			if moreSpecific(left, right) || moreSpecific(right, left) {
				continue
			}
			if left.Produces != right.Produces {
				continue
			}
			return fmt.Errorf(
				"routes %s %s and %s %s can match the same path and nothing separates them",
				left.Method, left.Pattern, right.Method, right.Pattern)
		}
	}
	return nil
}

func canCollide(left, right compiledRoute) bool {
	if len(left.segments) != len(right.segments) {
		return false
	}
	for i := range left.segments {
		l, r := left.segments[i], right.segments[i]
		if !l.isCapture() && !r.isCapture() && l.literal != r.literal {
			return false
		}
	}
	return true
}

// accepts decides Spring's produces condition. An operation that promises
// nothing serves any Accept, and an absent Accept accepts anything.
func accepts(header, produces string) bool {
	if produces == "" || strings.TrimSpace(header) == "" {
		return true
	}
	produced := strings.SplitN(produces, "/", 2)
	for _, entry := range strings.Split(header, ",") {
		// Quality values do not change whether a type is acceptable at all,
		// only which of several is preferred, and each operation here produces
		// exactly one type.
		media := strings.TrimSpace(strings.SplitN(entry, ";", 2)[0])
		if media == "*/*" || media == produces {
			return true
		}
		if len(produced) == 2 && media == produced[0]+"/*" {
			return true
		}
	}
	return false
}

func pathSegments(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}
