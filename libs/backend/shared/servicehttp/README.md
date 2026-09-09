# Go Shared Service HTTP

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

The HTTP plumbing `identity-service` and `profile-service` serve through:
request routing, the CORS layer and the request-duration metric. One
implementation, two consumers, for the same reason as
[`backend-shared-auth`](../auth) — the two services were brought into measured
agreement with the Spring `usersrole` application, and three files they each
held their own copy of are three files that could drift apart afterwards
without either test suite noticing.

Nothing here is configurable beyond what a service genuinely owns. A service
passes its **route table** and its **CORS configuration**; the resolution
order, the refusal shapes, the metric name and its label set are the same for
both by construction.

---

## 📁 Project Structure

```
libs/backend/shared/servicehttp/
├── go.mod                # module libs/backend/shared/servicehttp
├── router.go             # Spring's path specificity and refusal ordering
├── cors.go               # The CorsFilter SecurityConfig installs, reproduced
├── metrics.go            # http_server_requests_seconds, as Micrometer names it
└── project.json          # Nx project configuration
```

---

## 🧭 Routing: Spring's resolution, not ServeMux's

`Router` resolves a request the way `RequestMappingHandlerMapping` resolves it,
because the Go services and the JVM have to agree on every path where more than
one pattern matches.

```go
import "libs/backend/shared/servicehttp"

router, err := servicehttp.NewRouter([]servicehttp.Route{
  {Method: http.MethodGet, Pattern: "/api/profiles/{profileId}", Handler: byID},
  {Method: http.MethodGet, Pattern: "/api/profiles/{profileId}/icon", Produces: "image/png", Handler: icon},
})
```

`net/http`'s `ServeMux` is not a drop-in for this on two counts. It cleans and
redirects paths of its own accord — `//api/users` answers 301 where Spring
routes it, so a client that built a URL by joining strings would start being
redirected at cutover instead of served. And it refuses at registration any two
patterns where neither matches a strict subset of the other, which is exactly
the shape `/api/profiles/by-user/{userId}` and `/api/profiles/{profileId}/icon`
make: registering the profile surface on a `ServeMux` panics before the service
can start.

What the router reproduces, each pinned by a test here and by each service's
measured-routing suite against its own contract:

| Behaviour              | Rule transcribed                                                             |
| ---------------------- | ---------------------------------------------------------------------------- |
| Pattern precedence     | `PathPattern.SPECIFICITY_COMPARATOR` — fewer captures wins, then longer      |
| Refusal ordering       | `handleNoMatch` — 404 before 405 before 406                                  |
| `Allow` on a 405       | the methods the path maps, sorted                                            |
| `produces` negotiation | `*/*`, `type/*` and an absent `Accept` all serve; anything else is 406       |
| Ambiguity              | refused at construction, where Spring throws at request time and answers 500 |

**Refusing an ambiguous pair at construction is deliberate.** Spring discovers
such a pair on the request that hits it, throwing `IllegalStateException:
Ambiguous handler methods mapped` and answering 500 to whoever asked. Failing to
start is a failure a deployment notices and a user never sees.

A refusal is a status the _container_ sets rather than one a handler composed,
so the router answers it through `authhttp.WriteContainerError` — Boot's error
JSON for a caller whose token survives the forward to `/error`, and the empty
401 for one whose forward is refused a second time. That rule, and the
measurements behind it, are in the error-body section of
[`backend-shared-auth`'s README](../auth/README.md).

---

## 🌐 CORS sits outside authentication

`CORS` reproduces the `CorsFilter` that `SecurityConfig` installs ahead of the
JWT filter, with the origin patterns, methods and headers its
`CorsConfigurationSource` registers for `"/**"`.

```go
handler := config.CORS.Handler(next)
```

**It must be the outer layer**, matching the JVM's filter order. A browser
never puts an `Authorization` header on a preflight, so a preflight that reached
authentication would be refused and every cross-origin call from the frontends
would fail at cutover with the request itself perfectly valid. The shared
authentication middleware emits no CORS headers of its own precisely so this
layer owns them.

`AllowedOriginPatterns` takes Spring's syntax rather than a glob: `*` matches
any run of characters within the origin, and a trailing `:[...]` names the
ports, with `[*]` standing for any port or none.

---

## 📈 Metrics keep the JVM's series name and labels

`Metrics` publishes `http_server_requests_seconds` with Micrometer's
`method`/`uri`/`status`/`outcome` labels, so the existing p50/p95/p99 panels
keep working across the cutover instead of going blank the moment traffic
moves. `RequestDurationName` is exported for the suites that assert a scrape
carries it.

```go
metrics := servicehttp.NewMetrics()
handler := metrics.Middleware(router, authenticated)
// and the scrape endpoint the JVM serves at the same path
routes = append(routes, servicehttp.Route{
  Method: http.MethodGet, Pattern: "/actuator/prometheus", Handler: metrics.Handler(),
})
```

Two things are worth knowing rather than discovering:

- **The `uri` label is the route pattern, never the request path.** Every id
  would otherwise open its own time series, and a caller-supplied segment would
  reach the exposition format — a scanner could write label values by probing.
  A request that matched no route is labelled `UNKNOWN`.
- **The bucket edges are explicit and are not the JVM's.** Micrometer generates
  its histogram from its own percentile configuration and it cannot be
  reproduced edge for edge, so a quantile a dashboard interpolates is close
  rather than identical across the cutover.

Each `Metrics` owns its registry rather than the default one, so two servers in
one process — which every test binary has — cannot collide on registration.

---

## 🧪 Testing

```bash
nx test backend-shared-servicehttp      # go test -race ./...
nx lint backend-shared-servicehttp      # go vet ./...
nx tidy backend-shared-servicehttp
```

The suite here uses a route table that is no service's, because what belongs
here is the resolution the two of them share. Each service keeps its own
routing tests: those cases are transcribed from `x-path-precedence` in its
frozen contract, resolved against a running `RequestMappingHandlerMapping`, and
they are evidence about that service's surface rather than about this router.

---

## 📌 Notes

- `lint` runs `go vet` rather than the executor's default `go fmt`, which
  rewrites files and never fails, so it would gate nothing. This matches
  `backend-shared-auth`.
- The only external dependency is `github.com/prometheus/client_golang`. The
  import of `libs/backend/shared/auth/authhttp` resolves through `go.work`, as
  it does for the services.

---

## 📚 Related Packages

- [`backend-shared-auth`](../auth): verification, the authorization rules, and
  the `WriteContainerError` this router answers a refusal with.
- [`backend-shared-util`](../util): the other Go library shared across services.
- [`identity-service`](../../../../apps/backend/identity-service) and
  [`profile-service`](../../../../apps/backend/profile-service): the two
  consumers.
- [`usersrole`](../../../../apps/backend/usersrole): the Spring service whose
  behaviour this reproduces, and the home of the frozen contracts.
