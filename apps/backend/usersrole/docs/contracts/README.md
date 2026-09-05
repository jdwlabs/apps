# Frozen service contracts

Two OpenAPI 3.1 documents that fix the API of the two Go services replacing this
Spring Boot application:

| File                            | Service            | Operations | Surface                                           |
| ------------------------------- | ------------------ | ---------- | ------------------------------------------------- |
| `identity-service.openapi.yaml` | `identity-service` | 18         | `/auth` (2), `/api/users` (8), `/api/roles` (8)   |
| `profile-service.openapi.yaml`  | `profile-service`  | 15         | `/api/profiles` and its address and icon subtrees |

33 operations, each appearing in exactly one document. The boundary is the one
the access-pattern audit froze; these files add the wire detail the audit did not
carry: per-operation authorization, the full error set, media types, pagination,
and the JWT claim contract the two services share.

They exist so a Go service can be built and verified against a written contract
instead of against Java, and so an authorization regression during the rewrite is
caught by a parity test rather than by a user. Each operation carries an
`x-authorization` block holding the exact `@PreAuthorize` predicate it was
transcribed from, so the parity suite can be generated from the specification
rather than re-derived from the source.

## How the documents were produced

They were extracted, not written. `usersrole` publishes a springdoc document at
`/openapi/api-docs`; a test boots the application on a random port, fetches that
document and writes it to `build/openapi/served-api-docs.json`:

```bash
cd apps/backend/usersrole
bash gradlew test --tests com.jdw.usersrole.contracts.ServedOpenApiDocumentDumpTests
```

Nothing was fetched from a deployed environment. The dump is a build artifact,
deliberately not committed: it is evidence for the diff, not a source of truth.

The served document was then split along the boundary and hand-corrected against
the four controllers and `GlobalExceptionHandler`, because springdoc reads
declared return types rather than the responses the handlers build. It reports
every operation as `200 */*`, documents no error response at all, and types both
icon uploads as `application/json`. Everything it gets wrong is listed in
`served-document-deviations.json` with the reason, and the drift check fails on
any difference that file does not account for.

## Checks

```bash
npx nx run usersrole:lint               # schema validation, both documents
npx nx run usersrole:check-contracts    # extract the served document, then diff against it
```

The `lint` target runs Redocly against `redocly.yaml`, which names the ruleset
rather than relying on the CLI default so a Redocly upgrade cannot quietly change
what CI enforces on a frozen document.

`check-contracts` re-extracts the served document and runs
`scripts/check-contract-drift.mjs`, which fails when:

- an operation is served but frozen in neither document, or frozen but not
  served, or frozen in both;
- either document's operation count moves off 18 or 15;
- a success status, request media type or parameter set differs from the served
  document in a way `served-document-deviations.json` does not record;
- an entry in that file no longer applies — so the ledger cannot rot into a list
  of excuses for things that were fixed years ago;
- any schema in either document, or in the served document, carries a `password`
  property that is not `writeOnly`. That one is asserted directly and no recorded
  deviation can excuse it.

Both run in CI, chosen by `nx affected`, so a change that does not touch this
project costs nothing. `lint` rides the existing `nx affected -t lint test` step.
`check-contracts` gets its own step, because it boots the application to fetch
the document it diffs against and `lint` everywhere else in this repo is static.

The drift check cannot see authorization: the served springdoc document carries
none. Two tests cover that half.

`AuthorizationContractParityTests` reflects over the four controllers, rebuilds
each mapping's path and HTTP method, and asserts that every operation's
`x-authorization.preAuthorize` is exactly the `@PreAuthorize` predicate the
handler applies. An operation that carries no predicate must say so with an
explicit `preAuthorize: null`, and a missing key fails — otherwise the six
operations without one would be asserted vacuously. It also fails if a controller
maps something no contract freezes, if a contract freezes something no controller
maps, or if the mapping count moves off 33. No booted application needed.

`FilterChainContractParityTests` covers what the annotations cannot say. The six
operations with no predicate are separated into PUBLIC and AUTHENTICATED by
`SecurityConfig`'s `.requestMatchers("/auth/**", "/actuator/**", "/openapi/**")
.permitAll()`, and nothing else in the build reads that matcher list — flipping
it would turn the public endpoints private with every other check still green. So
rather than parse the matcher list, which would assert the source against itself,
this drives real unauthenticated requests over all 33 operations and asserts the
outcome the contract claims: an operation frozen as PUBLIC must not answer 401,
every other operation must, and nothing outside `/auth` may be frozen as PUBLIC.

## The decisions this contract settles

The audit deferred seven contract questions. Each is answered here, and each
answer also lives on the operation or schema it governs, so an implementer
reading the specification alone finds it.

### 1. The identity user payload drops the embedded profile

`User` exposes `profileId` instead of `profile`.

Today the `User` payload embeds the whole profile aggregate — the profile record,
its address set and its icon. Keeping it would force `identity-service` to call
`profile-service` on every user read: a synchronous hop on the hottest path in
the system, to serve data the caller usually does not want, and the one outcome
the split must not produce. Dropping it also removes 3 of the 4 statements in the
audit's per-row multiplier, taking a default-size `GET /api/users` from about 401
statements to 101 before any batching.

No current client reads `user.profile` — the profile screens call
`profiles.service.ts`, which already addresses `/api/profiles` directly. Clients
that need the profile fetch `GET /api/profiles/user/{userId}`.

Recorded as `x-behaviour-change` on `identity-service:User`.

### 2. The password is write-only on input and absent from every response

`UserRequestDTO.password` is `writeOnly`; no response schema in either document
carries a password field.

This was a live defect when the audit was written — every `User` response
serialized the bcrypt hash, including the public registration endpoint. It has
since been closed in the application: `User.password` is `@JsonIgnore`, and a
guard test scans every model and DTO for password-like fields left serializable.
The contract's job is to keep it closed through the rewrite, so the drift check
asserts the property against both the frozen documents and the served one, and no
recorded deviation can excuse a failure.

### 3. The icon is identified by `profileId`

`ProfileIconDaoPostgres` changes its mind about what "the id" means between
methods, four times over: `findById` filters on `profile_id` while being named as
if it took an icon id; `update` and `deleteById` filter on `icon_id`; `create`
returns `profile_id`, which `deleteById` would misread; and `update` filters
correctly on `icon_id` and then returns `findByProfileId(profileIcon.id())`,
feeding an icon id to a `profile_id` filter.

That last one is live code, not a latent trap — `ProfileRepositoryImpl.saveIcon`
calls both `create` and `update`. It is invisible today only because `saveIcon`
throws away what the DAO returns and re-reads the profile, and because the two
identity sequences still happen to agree in the deployed data.

**`profile_id` is the icon's only identifier.** Every route already addresses the
icon as `/api/profiles/{profileId}/icon` and none accepts an icon id; the choice
costs nothing, matches the routes, and matches the natural key, since a profile
holds at most one icon. Choosing `icon_id` instead would mean inventing routes
that do not exist and changing every client.

That "at most one" is enforced by the application, not the schema:
`profile_icons_profile_id_idx` is a plain index, and only
`ProfileService.addIcon` stops a second row by reading first and answering 409.
Making it unique is the right end state and is a migration, so it belongs to its
own change; until then the Go service keeps the application-level check rather
than assuming the database holds the line.

`ProfileIcon.id` stays in the response, because a client type declares it, and is
documented as an opaque surrogate: no route accepts it, no lookup uses it, and
the Go data access must not key on it.

Recorded as `x-icon-identifier` in the profile contract.

### 4. `GET /api/roles` is paginated — a behaviour change

The endpoint is unpaginated today: `SELECT * FROM auth.roles` with no `ORDER BY`
and no `LIMIT`, returning the whole table unordered, with a fan-out the audit
measured at `1 + N` statements over that table.

The contract gives it `page` and `size` with the same defaults, 0 and 100, and
the same bare-array response as the user and profile listings, over
`ORDER BY role_id`. The ordering is part of the change: pagination without a
total order returns overlapping pages.

**Client impact.** The three role-list callers in the frontends pass no query
parameters, so they receive the first 100 roles rather than all of them. The
deployed catalogue holds 3 rows, so nothing changes today; a workspace past 100
roles would notice, and recording it here is the point.

Recorded as `x-behaviour-change` on `GET /api/roles`.

### 5. Role enumeration stays open to any authenticated principal

The three role reads carry no method-level predicate today, so any authenticated
principal can enumerate the whole catalogue. **Frozen as-is, deliberately.**

Neither the container nor any remote applies a client-side role gate — a search
for role checks across `libs/frontend` and `apps/frontend` returns one unrelated
comment. The role screens are reachable by every authenticated user, and they are
reachable precisely because these three reads answer them. Tightening to
ADMIN-or-MANAGER turns those screens into 403s at cutover with no client change
to compensate, which is exactly the silent authorization regression this contract
exists to make impossible.

The exposure that actually matters is narrower than the catalogue: the `users`
array on each `Role` discloses which user ids hold which role. Narrowing that
needs a frontend change to land with it, because the two assignment screens
consume it, so it is out of scope here — recorded so it is not mistaken for an
oversight.

Recorded as `x-authorization-decision` on `GET /api/roles`.

### 6. A stale `profile_id` claim falls back to a `user_id` lookup

There is no lockout today. `SecurityUser.getProfileId()` walks the `User`
aggregate the filter loaded from the database this request, so a profile created
mid-session is visible on the very next request. The lockout is something the
split would introduce: once the ten operations that authorize on `profileId` read
it from a claim stamped at login, a user who logs in without a profile and then
creates one carries a null claim for up to the full two-hour token life.

**`profile-service` falls back to a `user_id`-keyed profile lookup whenever the
claim is absent or null.** `identity-service` does not re-mint tokens on profile
creation.

The fallback is one indexed read on `auth.profiles.user_id`, taken only when the
claim is missing; a present claim short-circuits it. Re-minting instead would
need `profile-service` to reach back into `identity-service` at write time — the
synchronous coupling the split exists to avoid — and would still leave every
already-issued token stale. The lookup keys on the `user_id` claim, never on a
path variable, so a caller cannot widen their own authorization by asking for
someone else's profile id.

A _stale non-null_ claim is a different case, and the fallback does not cover it:
a caller who deletes their own profile keeps a claim pointing at a row that is
gone, and the predicate then passes on all ten operations. Eight of them look the
resource up first and answer 404 where today they answer 403; the two that delete
without looking up — `DELETE /api/profiles/{profileId}` and
`DELETE /api/profiles/{profileId}/icon` — answer 204, a silent success for a
caller who owns nothing. Both are widenings and both are recorded, in
`x-stale-profile-claim` and on the delete operation itself.

Recorded as `x-profile-id-fallback` in the profile contract.

### 7. Deleting a user stays one local transaction in identity-service

`DELETE /api/users/{userId}` removes the user's role grants, its profile, and
that profile's addresses and icon. The last three tables belong to
`profile-service`, so this is the only write that crosses the boundary.

**It stays a single local transaction in `identity-service`: five explicit
deletes in a fixed order** — addresses by profile id, icons by profile id,
profiles by user id, role grants by user id, then the user. The alternative — a
synchronous call from identity into profile inside a delete — makes user deletion
fail whenever `profile-service` is down, and leaves a half-deleted user when it
fails midway.

**The database is not doing this.** No foreign key in `auth` declares
`ON DELETE CASCADE`; all five take the default `NO ACTION` — `users_roles`
carries two of them, one to each parent. The ordering is
therefore load-bearing application logic that the Go service has to reproduce,
not a database behaviour it inherits for free. Moving the cascade into the schema
is the better end state and is a migration, so it belongs to its own change
rather than to this freeze.

**The cost, recorded rather than hidden.** `profile-service` is not the only
writer of its own tables. It must not cache profile rows, and it cannot assume a
row it read still exists. The two services also stay bound to one schema and one
migration timeline. That is accepted here because the tables are already shared
by design and physical separation is out of scope for the split.

Recorded as `x-delete-cascade` on the delete operation.

## Frontend impact: no client change at cutover

All four API groups are addressed through one environment token. `auth`, `users`
and `roles` come from `libs/frontend/shared/data-access/src/lib/**`, profiles from
`libs/frontend/usersui/data-access/src/lib/profiles/profiles.service.ts`, and
every one of them builds its URL as `${this.environment.AUTH_BASE_URL}/api/...`
or `${...}/auth/...`. Keeping one hostname and splitting by path prefix is
therefore the zero-frontend-change option, and it is what both documents declare
as their server.

Two further findings support that claim:

- **No client reads an error body.** The shared helper
  `libs/frontend/shared/util/src/lib/http-error-message.util.ts` switches on the
  status code alone and returns a fixed string for 400, 401, 403, 404, 409 and 500. The error _bodies_ frozen here are therefore documentation for service
  authors, not a client dependency — but the _status codes_ are load-bearing, and
  changing one changes what a user reads.
- **No client reads `user.profile`.** Checked before dropping it; see decision 1.

## Everything these documents change

"Frozen" does not mean identical. Five behaviours differ from what runs today,
and each is marked on the operation or schema it affects so it cannot be mistaken
for a transcription.

| Change                                                                                                          | Where                                                                       | Class    | Visible to a client today?                                                   |
| --------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- | -------- | ---------------------------------------------------------------------------- |
| `User` drops the embedded profile for `profileId`                                                               | `identity-service:User`                                                     | design   | No — nothing reads `user.profile`                                            |
| `GET /api/roles` gains `page` and `size`                                                                        | `GET /api/roles`                                                            | design   | No — the catalogue holds 3 rows                                              |
| Role grants come from the token, not a per-request read                                                         | `x-authority-freshness`                                                     | security | Not today; after the split, up to 2 h of revocation latency                  |
| A stale `profile_id` claim yields 404 where today it is 403, on the eight profile operations that look up first | `x-stale-profile-claim`                                                     | security | Only after deleting your own profile — a different message, no broken screen |
| A stale `profile_id` claim yields 204 where today it is 403, on the two that delete without looking up          | `DELETE /api/profiles/{profileId}`, `DELETE /api/profiles/{profileId}/icon` | security | Only after deleting your own profile — a silent success instead of a refusal |

The last three follow from the split itself rather than from a choice made here:
`profile-service` will not own `auth.users` or `auth.users_roles`, so it cannot
keep re-reading them. They are listed because a reviewer comparing the Go service
against this contract must not read "authorizes from the verified token" as
equivalent to today's behaviour. It is not, and where it differs it differs in
the permissive direction.

### A correction that has since become a transcription

`DELETE /api/profiles/{profileId}/address/{addressId}` is specified scoped to the
profile in the path, answering 404 when the address does not belong to it.

When this contract was written that was a deviation. The handler took
`profileId`, used it for the authorization check, and then deleted by `addressId`
alone, so any authenticated principal with a profile could delete any address in
the table by guessing a sequential id and get 204 for it. Transcribing that
faithfully would have written an insecure direct object reference into the
document the Go services are built and parity-tested against, and the generated
suite would have certified it — so the contract specified the scoped behaviour
and recorded the gap.

The application has since been fixed independently, in
`AddressDaoPostgres.deleteByProfileIdAndAddressId` and
`ProfileService.deleteAddress`. Contract and code now agree, and this is a
transcription like every other operation. The history stays here because it is
the clearest example of what these documents are for: the contract named the
behaviour that ought to hold before anything enforced it.
