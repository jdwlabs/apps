# usersrole Access-Pattern Audit and Frozen Service Boundary

Audit date: 2026-09-05. Source revision: `main` at `2c2cc18c`. Deployed image
under measurement: `docker.io/jdwlabs/usersrole:1.2.2`.

This document freezes the service boundary for the split of the usersrole
Spring Boot monolith into two Go services, `identity-service` and
`profile-service`. It records the endpoint inventory, the SQL every endpoint
issues, the DTO-to-table mapping, and a measured memory/CPU/latency baseline
for the production deployment.

Every number below is a measurement. Each table states how it was collected and
when. Where a measurement was not possible, the row says so and states what
would be needed instead — nothing is estimated silently.

## 1. How the numbers were collected

| Source                   | Method                                                                                                                         | Scope                             |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------ | --------------------------------- |
| Prometheus               | In-cluster query API, instant queries with range selectors over a 7-day and 30-day window. Exact PromQL recorded in section 7. | prd and non, read-only            |
| `pg_stat_user_tables`    | Snapshot / request / snapshot delta against the non Postgres cluster, over the local socket as the `postgres` role.            | non only, read-only catalog reads |
| `pg_indexes`, `count(*)` | Direct catalog and row-count reads.                                                                                            | non and prd, read-only            |
| Source                   | Static derivation from the four controllers, six DAOs and three repository implementations at revision `2c2cc18c`.             | n/a                               |

Production was touched only through Prometheus and two read-only catalog
queries. No load was generated against prd, no prd application request was
issued, and no cluster state was changed in either environment.

### 1.1 What could not be measured directly

`pg_stat_statements` is **not available** in either Postgres cluster: both run
with `shared_preload_libraries = ''` and only the `plpgsql` extension
installed. Loading it is a cluster-spec change plus a rolling restart, which is
outside a read-only audit. The exact change, for whoever wants per-statement
counts later:

```yaml
# CloudNativePG Cluster spec
spec:
  postgresql:
    shared_preload_libraries:
      - pg_stat_statements
```

followed by `CREATE EXTENSION pg_stat_statements;` in the target database and a
rolling restart of the instances.

Instead, fan-out was measured with `pg_stat_user_tables` scan-count deltas
around single requests (section 4). That counts one scan per statement
execution for these access paths — every DAO statement here touches exactly one
table with no joins — so a scan delta is a statement count.

The three list endpoints could not be called end-to-end: every path except
`/auth/**`, `/actuator/**` and `/openapi/**` requires an authenticated
principal, and minting an admin credential inside an automated session is
against this platform's credential-handling rule. What was measured instead is
the per-row hydration helper those endpoints call, which is the identical
method (`UserRepositoryImpl.getUser`), invoked once per result row. Section 4
gives the measured per-row cost and the arithmetic that follows from it, and
labels the list totals as derived-from-measurement rather than observed.

## 2. Endpoint inventory

33 mappings across four controllers. The authorization column is the exact
method-level `@PreAuthorize` predicate; `(chain only)` means the method carries
no annotation and is governed solely by the filter chain rule
`.requestMatchers("/auth/**", "/actuator/**", "/openapi/**").permitAll().anyRequest().authenticated()`.

The traffic column is the measured 30-day request count in prd. The p95 column
is the measured 7-day p95; `no samples` means the histogram has no observations
in the window, so no percentile exists.

### 2.1 AuthController — 2 endpoints, bounded context: identity

| Method | Path                 | Authorization        | 30d requests (prd) | 7d p95     |
| ------ | -------------------- | -------------------- | ------------------ | ---------- |
| POST   | `/auth/authenticate` | (chain only, public) | 1 (200)            | no samples |
| POST   | `/auth/user`         | (chain only, public) | 0                  | no samples |

### 2.2 UsersController — 8 endpoints, bounded context: identity

| Method | Path                               | Authorization                                                                      | 30d requests (prd) | 7d p95     |
| ------ | ---------------------------------- | ---------------------------------------------------------------------------------- | ------------------ | ---------- |
| GET    | `/api/users`                       | `hasAuthority('ADMIN')`                                                            | 0                  | no samples |
| GET    | `/api/users/{userId}`              | `hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()`         | 1                  | no samples |
| GET    | `/api/users/email/{emailAddress}`  | `hasAuthority('ADMIN') or #emailAddress == authentication.principal.getUsername()` | 0                  | no samples |
| POST   | `/api/users`                       | (chain only, authenticated)                                                        | 0                  | no samples |
| PUT    | `/api/users/{userId}`              | `hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()`         | 0                  | no samples |
| DELETE | `/api/users/{userId}`              | `hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()`         | 0                  | no samples |
| PUT    | `/api/users/{userId}/roles/grant`  | `hasAnyAuthority('ADMIN', 'MANAGER')`                                              | 0                  | no samples |
| PUT    | `/api/users/{userId}/roles/revoke` | `hasAnyAuthority('ADMIN', 'MANAGER')`                                              | 0                  | no samples |

### 2.3 RolesController — 8 endpoints, bounded context: identity

| Method | Path                               | Authorization                         | 30d requests (prd) | 7d p95     |
| ------ | ---------------------------------- | ------------------------------------- | ------------------ | ---------- |
| GET    | `/api/roles`                       | (chain only, authenticated)           | 0                  | no samples |
| GET    | `/api/roles/{roleId}`              | (chain only, authenticated)           | 0                  | no samples |
| GET    | `/api/roles/name/{roleName}`       | (chain only, authenticated)           | 0                  | no samples |
| POST   | `/api/roles`                       | `hasAuthority('ADMIN')`               | 0                  | no samples |
| PUT    | `/api/roles/{roleId}`              | `hasAuthority('ADMIN')`               | 0                  | no samples |
| DELETE | `/api/roles/{roleId}`              | `hasAuthority('ADMIN')`               | 0                  | no samples |
| PUT    | `/api/roles/{roleId}/users/grant`  | `hasAnyAuthority('ADMIN', 'MANAGER')` | 0                  | no samples |
| PUT    | `/api/roles/{roleId}/users/revoke` | `hasAnyAuthority('ADMIN', 'MANAGER')` | 0                  | no samples |

The three read endpoints carry no method-level predicate: **any authenticated
principal can enumerate every role**. That is wider than the write endpoints on
the same resource and should be tightened deliberately in the Go contract
rather than carried across by accident.

### 2.4 ProfilesController — 15 endpoints, bounded context: profile

| Method | Path                                            | Authorization                                                                        | 30d requests (prd) | 7d p95     |
| ------ | ----------------------------------------------- | ------------------------------------------------------------------------------------ | ------------------ | ---------- |
| GET    | `/api/profiles`                                 | `hasAuthority('ADMIN')`                                                              | 0                  | no samples |
| GET    | `/api/profiles/{profileId}`                     | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| GET    | `/api/profiles/user/{userId}`                   | `hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()`           | 0                  | no samples |
| POST   | `/api/profiles`                                 | `hasAuthority('ADMIN') or #profile.userId() == authentication.principal.getUserId()` | 0                  | no samples |
| PUT    | `/api/profiles/{profileId}`                     | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| PUT    | `/api/profiles/user/{userId}`                   | `hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()`           | 0                  | no samples |
| DELETE | `/api/profiles/{profileId}`                     | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| DELETE | `/api/profiles/user/{userId}`                   | `hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()`           | 0                  | no samples |
| POST   | `/api/profiles/{profileId}/address`             | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| PUT    | `/api/profiles/{profileId}/address/{addressId}` | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| DELETE | `/api/profiles/{profileId}/address/{addressId}` | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| GET    | `/api/profiles/{profileId}/icon`                | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| POST   | `/api/profiles/{profileId}/icon`                | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| PUT    | `/api/profiles/{profileId}/icon`                | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |
| DELETE | `/api/profiles/{profileId}/icon`                | `hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()`     | 0                  | no samples |

### 2.5 Measured traffic share

Over the 7 days ending 2026-09-05, prd served **no business traffic at all**.
The complete request mix:

| URI                       | Method | Status | 7d count | Share  |
| ------------------------- | ------ | ------ | -------- | ------ |
| `/actuator/health/**`     | GET    | 200    | 120,960  | 85.22% |
| `/actuator/prometheus`    | GET    | 200    | 20,160   | 14.20% |
| `UNKNOWN`                 | GET    | 401    | 813      | 0.57%  |
| `UNKNOWN`                 | POST   | 401    | 9        | 0.006% |
| `/**`                     | GET    | 404    | 2        | 0.001% |
| all `/api/**`, `/auth/**` | —      | —      | **0**    | **0%** |

The two actuator figures are exactly the probe and scrape schedules
(120,960 = 7 d ÷ 5 s; 20,160 = 7 d ÷ 30 s), which confirms the counters are
sound rather than stalled. The `UNKNOWN` 401s are unauthenticated internet
scanner noise rejected by the filter chain before routing.

Widening to 30 days finds **two** real business requests in total: one
`POST /auth/authenticate` and one `GET /api/users/{userId}`.

**Consequence for the sizing work downstream.** No per-endpoint p50/p95/p99 can
be reported for any business endpoint, because the histograms have no
observations. This is not a gap in the instrumentation — `percentiles-histogram`
is enabled for `http.server.requests` in `application.yaml` and the chart ships
a ServiceMonitor scraping every 30 s, and both are confirmed working by the
actuator percentiles in section 6.3. It is an absence of load. Any latency
target for the Go services must therefore be set from a synthetic benchmark run
against a seeded dataset, not from production history.

## 3. Query shapes

40 SQL literals across six Postgres DAOs. All statements are single-table; there
are no joins anywhere in the codebase. `idx` / `seq` records the access path
observed in `pg_stat_user_tables` (section 4) or implied by the index inventory
in section 3.7.

### 3.1 UserDaoPostgres — 6 statements, table `auth.users`

| Repository method    | SQL                                                                                                                                                             | Path |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---- |
| `create`             | `INSERT INTO auth.users (email_address, password, status, created_by_user_id, created_time, modified_by_user_id, modified_time) VALUES (...) RETURNING user_id` | —    |
| `findById`           | `SELECT * FROM auth.users WHERE user_id = :id`                                                                                                                  | idx  |
| `findByEmailAddress` | `SELECT * FROM auth.users WHERE email_address = :emailAddress`                                                                                                  | idx  |
| `findAll`            | `SELECT * FROM auth.users ORDER BY user_id LIMIT :limit OFFSET :offset`                                                                                         | idx  |
| `update`             | `UPDATE auth.users SET email_address=…, password=…, status=…, modified_by_user_id=…, modified_time=… WHERE user_id = :id`                                       | idx  |
| `deleteById`         | `DELETE FROM auth.users WHERE user_id = :id`                                                                                                                    | idx  |

`create` and `update` each issue a follow-up `findById`, so they cost two
statements at the DAO level before repository hydration is counted.

### 3.2 RoleDaoPostgres — 6 statements, table `auth.roles`

| Repository method | SQL                                                                                                                           | Path |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------- | ---- |
| `create`          | `INSERT INTO auth.roles (role_name, role_description, status, …) VALUES (…) RETURNING auth.roles.role_id`                     | —    |
| `findById`        | `SELECT * FROM auth.roles WHERE role_id = :id`                                                                                | idx  |
| `findByName`      | `SELECT * FROM auth.roles WHERE role_name = :name`                                                                            | seq  |
| `findAll`         | `SELECT * FROM auth.roles`                                                                                                    | seq  |
| `update`          | `UPDATE auth.roles SET role_name=…, role_description=…, status=…, modified_by_user_id=…, modified_time=… WHERE role_id = :id` | idx  |
| `deleteById`      | `DELETE FROM auth.roles WHERE role_id = :id`                                                                                  | idx  |

`findAll` has no `ORDER BY` and no `LIMIT`; it returns the whole table
unordered. Unlike the user and profile listings it is not paginated at all.

### 3.3 UserRoleDaoPostgres — 7 statements, table `auth.users_roles`

| Repository method         | SQL                                                                                            | Path |
| ------------------------- | ---------------------------------------------------------------------------------------------- | ---- |
| `create`                  | `INSERT INTO auth.users_roles (user_id, role_id, created_by_user_id, created_time) VALUES (…)` | —    |
| `findByRoleId`            | `SELECT * FROM auth.users_roles WHERE role_id = :roleId`                                       | seq  |
| `findByUserId`            | `SELECT * FROM auth.users_roles WHERE user_id = :userId`                                       | idx  |
| `findByRoleIdAndUserId`   | `SELECT * FROM auth.users_roles WHERE role_id = :roleId AND user_id = :userId`                 | idx  |
| `deleteByRoleId`          | `DELETE FROM auth.users_roles WHERE role_id = :roleId`                                         | seq  |
| `deleteByUserId`          | `DELETE FROM auth.users_roles WHERE user_id = :userId`                                         | idx  |
| `deleteByRoleIdAndUserId` | `DELETE FROM auth.users_roles WHERE role_id = :roleId AND user_id = :userId`                   | idx  |

The primary key is `(user_id, role_id)` in that order, so `user_id` predicates
use the index and `role_id` predicates cannot. This was confirmed by observation,
not inferred: section 4.3.

### 3.4 ProfileDaoPostgres — 7 statements, table `auth.profiles`

| Repository method | SQL                                                                                                                     | Path |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------- | ---- |
| `create`          | `INSERT INTO auth.profiles (user_id, first_name, middle_name, last_name, birthdate, …) VALUES (…) RETURNING profile_id` | —    |
| `findById`        | `SELECT * FROM auth.profiles WHERE profile_id = :id`                                                                    | idx  |
| `findByUserId`    | `SELECT * FROM auth.profiles WHERE user_id = :id`                                                                       | seq  |
| `findAll`         | `SELECT * FROM auth.profiles ORDER BY profile_id LIMIT :limit OFFSET :offset`                                           | idx  |
| `update`          | `UPDATE auth.profiles SET first_name=…, … WHERE profile_id = :id`                                                       | idx  |
| `deleteById`      | `DELETE FROM auth.profiles WHERE profile_id = :id`                                                                      | idx  |
| `deleteByUserId`  | `DELETE FROM auth.profiles WHERE user_id = :id`                                                                         | seq  |

### 3.5 AddressDaoPostgres — 7 statements, table `auth.addresses`

| Repository method   | SQL                                                                                                                                                      | Path |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ---- |
| `create`            | `INSERT INTO auth.addresses (profile_id, address_line_1, address_line_2, city, state_province, postal_code, country, …) VALUES (…) RETURNING address_id` | —    |
| `findById`          | `SELECT * FROM auth.addresses WHERE address_id = :id`                                                                                                    | idx  |
| `findByProfileId`   | `SELECT * FROM auth.addresses WHERE profile_id = :id`                                                                                                    | seq  |
| `findAll`           | `SELECT * FROM auth.addresses`                                                                                                                           | seq  |
| `update`            | `UPDATE auth.addresses SET address_line_1=…, … WHERE address_id = :id`                                                                                   | idx  |
| `deleteById`        | `DELETE FROM auth.addresses WHERE address_id = :id`                                                                                                      | idx  |
| `deleteByProfileId` | `DELETE FROM auth.addresses WHERE profile_id = :id`                                                                                                      | seq  |

### 3.6 ProfileIconDaoPostgres — 7 statements, table `auth.profile_icons`

| Repository method   | SQL                                                                                    | Path |
| ------------------- | -------------------------------------------------------------------------------------- | ---- |
| `create`            | `INSERT INTO auth.profile_icons (profile_id, icon, …) VALUES (…) RETURNING profile_id` | —    |
| `findById`          | `SELECT * FROM auth.profile_icons WHERE profile_id = :id`                              | seq  |
| `findByProfileId`   | `SELECT * FROM auth.profile_icons WHERE profile_id = :id`                              | seq  |
| `findAll`           | `SELECT * FROM auth.profile_icons`                                                     | seq  |
| `update`            | `UPDATE auth.profile_icons SET icon=…, … WHERE icon_id = :id`                          | idx  |
| `deleteById`        | `DELETE FROM auth.profile_icons WHERE icon_id = :id`                                   | idx  |
| `deleteByProfileId` | `DELETE FROM auth.profile_icons WHERE profile_id = :id`                                | seq  |

This DAO has an identifier inconsistency the Go rewrite must resolve rather
than reproduce. `findById` filters on `profile_id` while being named and called
as if it took an icon id; it is byte-for-byte the same statement as
`findByProfileId`. Meanwhile `update` and `deleteById` filter on `icon_id`, and
`create` returns `profile_id`. So the DAO's notion of "the id" changes between
methods, and `create` hands back a value that `deleteById` would misinterpret.
Nothing in the current call graph trips over this — `ProfileRepositoryImpl` only
ever calls `findByProfileId` and `deleteByProfileId` — but the contract work
should pick one identifier for the icon resource and keep it.

### 3.7 Index inventory and the missing indexes

Measured from `pg_indexes` in the non cluster on 2026-09-05:

| Table                | Indexes                                                           |
| -------------------- | ----------------------------------------------------------------- |
| `auth.users`         | `users_pkey (user_id)`, `users_email_address_key (email_address)` |
| `auth.roles`         | `roles_pkey (role_id)`                                            |
| `auth.users_roles`   | `users_roles_pkey (user_id, role_id)`                             |
| `auth.profiles`      | `profiles_pkey (profile_id)`                                      |
| `auth.addresses`     | `addresses_pkey (address_id)`                                     |
| `auth.profile_icons` | `profile_icons_pkey (icon_id)`                                    |

**Every foreign-key column that the hydration path filters on is unindexed.**
There is no index on `auth.profiles.user_id`, `auth.addresses.profile_id`,
`auth.profile_icons.profile_id`, or the trailing `role_id` of
`auth.users_roles`. This was confirmed by observation: the fan-out runs in
section 4 incremented `seq_scan` and never `idx_scan` on exactly those tables.

At current row counts this costs nothing measurable — the largest table holds 9
rows. It matters because it makes the fan-out cost in section 4 quadratic in
table size rather than linear, so the N+1 shape and the missing indexes have to
be fixed together in the Go data-access design. Fixing only the N+1 leaves
`WHERE role_id = ?` on a growing join table scanning the whole table.

## 4. Fan-out: measured queries per request

### 4.1 Method

Snapshot `pg_stat_user_tables` for the `auth` schema in non, issue exactly one
HTTP request, wait 4 s for the statistics collector to flush, snapshot again,
and difference `seq_scan + idx_scan` per table. All statements in section 3 are
single-table and join-free, so one scan is one statement execution.

The request used is `POST /auth/authenticate` with a correctly-formatted but
deliberately wrong password. It returns 401 and writes nothing, and it drives
`JwtUserDetailService.loadUserByUsername` → `UserRepositoryImpl.findByEmailAddress`
→ the private `getUser` helper. That helper is the exact method the user listing
applies to every row, which is what makes these numbers transferable to the
list endpoints.

Non-prod fixture at time of measurement: 3 users, 3 roles, 9 user-role links, 2
profiles, 1 address, 1 icon. User 1 has a profile with one address and one icon;
user 3 has no profile row.

### 4.2 Measured per-row hydration cost

| Case                                          | users | users_roles | profiles | addresses | profile_icons | **Total** |
| --------------------------------------------- | ----- | ----------- | -------- | --------- | ------------- | --------- |
| User with profile, address, icon              | 1 idx | 1 idx       | 1 seq    | 1 seq     | 1 seq         | **5**     |
| User with profile, address, icon (repeat run) | 1 idx | 1 idx       | 1 seq    | 1 seq     | 1 seq         | **5**     |
| User with no profile row                      | 1 idx | 1 idx       | 1 seq    | —         | —             | **3**     |
| Email not present (control)                   | 1 idx | —           | —        | —         | —             | **1**     |
| Malformed request body (control)              | —     | —           | —        | —         | —             | **0**     |

The two identical runs establish reproducibility; the two controls establish
that the harness attributes scans to the request under test and not to
background activity.

### 4.3 Access paths, confirmed by observation

The delta table above is also the evidence for section 3.7. `auth.users` and
`auth.users_roles` incremented `idx_scan`, matching `users_email_address_key`
and the `user_id` prefix of `users_roles_pkey`. `auth.profiles`,
`auth.addresses` and `auth.profile_icons` incremented `seq_scan`, because none
of them has an index on the filtered column.

Corroborating this independently, the cumulative counters carry
`auth.roles.idx_scan = 18` against `auth.users_roles.seq_scan = 18` — an exact
1:1 correspondence produced by `SecurityUser.getAuthorities()`, which calls
`RoleRepository.findById` (indexed on `roles`) and thence
`UserRoleDao.findByRoleId` (unindexed on `users_roles`) once per granted role.

### 4.4 Per-request totals for the three list endpoints

Derived from the measured per-row cost, not observed end-to-end. The
multiplicand is measured; the multiplier is the page size.

`UsersController` and `ProfilesController` both default `size` to 100.

| Endpoint            | Statements                                                    | At `size=100`        |
| ------------------- | ------------------------------------------------------------- | -------------------- |
| `GET /api/users`    | 1 listing + 5 per row with a profile, 3 per row without       | **301 – 501**        |
| `GET /api/profiles` | 1 listing + 2 per row (`addresses`, `profile_icons`)          | **201**              |
| `GET /api/roles`    | 1 listing + 1 per row (`users_roles` by `role_id`, unindexed) | **1 + N**, unbounded |

`GET /api/roles` has no pagination at all, so its multiplier is the whole
`auth.roles` table rather than a page size.

### 4.5 The fixed preamble on every authenticated request

`JwtAuthenticationFilter` calls `loadUserByUsername` on every request carrying a
bearer token, which runs the same hydration measured in 4.2. Evaluating a
`@PreAuthorize` predicate then calls `SecurityUser.getAuthorities()`, which
costs 2 further statements per granted role — `roles` by id, then `users_roles`
by `role_id`.

For a principal holding 3 roles, as all three non-prod users do, that is a
floor of **5 + 6 = 11 statements before the handler body executes**, on every
authenticated request including ones that return 403. A default-page
`GET /api/users` by such a principal therefore issues on the order of
**512 statements**.

This preamble is the single most important thing for the Go design to change.
It is per-request work that the JWT already carries the answer to: the token
minted by `JwtService` contains `user_id`, `profile_id` and `roles` claims, yet
the filter re-reads all three from Postgres on every request.

## 5. DTO and entity mapping

### 5.1 Request DTOs

| DTO                       | Fields                                                                           | Target table     | Column mapping                                                                         |
| ------------------------- | -------------------------------------------------------------------------------- | ---------------- | -------------------------------------------------------------------------------------- |
| `UserRequestDTO`          | `emailAddress`, `password`                                                       | `auth.users`     | `email_address`, `password` (bcrypt-encoded before write)                              |
| `RoleRequestDTO`          | `name`, `description`                                                            | `auth.roles`     | `role_name`, `role_description`                                                        |
| `ProfileCreateRequestDTO` | `firstName`, `middleName`, `lastName`, `birthdate`, `userId`                     | `auth.profiles`  | `first_name`, `middle_name`, `last_name`, `birthdate`, `user_id`                       |
| `ProfileUpdateRequestDTO` | `firstName`, `middleName`, `lastName`, `birthdate`                               | `auth.profiles`  | as above, minus `user_id` (immutable after create)                                     |
| `AddressRequestDTO`       | `addressLine1`, `addressLine2`, `city`, `stateProvince`, `postalCode`, `country` | `auth.addresses` | `address_line_1`, `address_line_2`, `city`, `state_province`, `postal_code`, `country` |
| `AuthResponseDTO`         | `jwtToken`                                                                       | none             | minted, not persisted                                                                  |

### 5.2 Response payloads

There is exactly one response DTO, `AuthResponseDTO`. **Every other endpoint
returns a model record directly** — `User`, `Role`, `Profile`, or a raw
`byte[]`. There is no separate read model and no serialization view.

| Endpoint group                        | Returns           | Assembled from                                                                        | Tables read                                                               |
| ------------------------------------- | ----------------- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `POST /auth/authenticate`             | `AuthResponseDTO` | `JwtService.generateToken` over a `SecurityUser`                                      | `users`, `users_roles`, `profiles`, `addresses`, `profile_icons`, `roles` |
| `POST /auth/user`, all `/api/users/*` | `User`            | `User` + `Set<UserRole>` + embedded `Profile` (with `Set<Address>` and `ProfileIcon`) | `users`, `users_roles`, `profiles`, `addresses`, `profile_icons`          |
| all `/api/roles/*`                    | `Role`            | `Role` + `Set<UserRole>`                                                              | `roles`, `users_roles`                                                    |
| all `/api/profiles/*` except icon GET | `Profile`         | `Profile` + `Set<Address>` + `ProfileIcon`                                            | `profiles`, `addresses`, `profile_icons`                                  |
| `GET /api/profiles/{profileId}/icon`  | `byte[]`          | `ProfileIcon.icon`, served as `image/png`                                             | `profile_icons`                                                           |

Two consequences follow directly from returning models rather than DTOs.

**The bcrypt password hash is serialized in every `User` response.** The `User`
record exposes a public `password()` accessor, and the codebase contains no
`@JsonIgnore`, no `@JsonProperty(access = WRITE_ONLY)`, no Jackson mixin and no
`ObjectMapper` customization anywhere under `src/main`. Every endpoint returning
`User` — including the public `POST /auth/user` — therefore emits the stored
hash to the client. The Go contract must define an explicit response type that
omits it; this is the clearest single argument for introducing real DTOs in the
rewrite.

**The `User` payload embeds the entire profile aggregate.** This is what makes
the user listing cost 5 statements per row instead of 2, and it is the coupling
that the service boundary has to sever. See section 6.2.

## 6. The frozen boundary

### 6.1 Decision

**Confirmed: two services, split as originally targeted.** The measurements
support the boundary rather than contradicting it, with one amendment to the
`User` response contract recorded in 6.2.

| Service            | Bounded contexts           | Endpoints                                            | Tables owned                                            |
| ------------------ | -------------------------- | ---------------------------------------------------- | ------------------------------------------------------- |
| `identity-service` | auth, users, roles         | 18 (`/auth/*` 2, `/api/users/*` 8, `/api/roles/*` 8) | `auth.users`, `auth.roles`, `auth.users_roles`          |
| `profile-service`  | profiles, addresses, icons | 15 (`/api/profiles/*`)                               | `auth.profiles`, `auth.addresses`, `auth.profile_icons` |

Both services keep the shared `auth` schema; physical schema separation stays
out of scope.

Evidence for the split, line by line:

- **The table sets are disjoint.** The six DAOs partition cleanly three-and-three
  along the proposed line. No statement in section 3 joins across it — there are
  no joins at all.
- **The endpoint sets are disjoint.** All 15 profile endpoints touch only the
  profile three; all 18 identity endpoints touch the identity three, except for
  the hydration path called out in 6.2.
- **Write ownership is disjoint.** Every `INSERT`, `UPDATE` and `DELETE` in
  section 3 is issued by the service that owns the target table, with the single
  exception of `UserRepositoryImpl.deleteById`, which cascades into
  `addresses`, `profile_icons` and `profiles` when a user is removed.
- **The load justifies exactly two processes and no more.** Measured 7-day CPU
  averages 0.83 millicores against a 500 m request, and HikariCP never showed a
  single active connection. A finer split would multiply fixed per-process
  overhead against a workload that does not exist.

Alternatives rejected:

| Option                                                   | Rejected because                                                                                                                                                                                                                                                 |
| -------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Five-way split (auth / users / roles / profiles / icons) | Measured load is two business requests in 30 days and a 7-day CPU average of 0.83 millicores. Five processes would add five fixed memory floors and four network hops to serve it.                                                                               |
| Keep the monolith, rewrite in Go as one service          | Does not address the coupling in 6.2, which is the actual defect: the identity payload embeds the profile aggregate. A single Go service would carry it across unchanged.                                                                                        |
| Split users from roles                                   | `auth.users_roles` is written by both `UserRepositoryImpl.grantRoles` and `RoleRepositoryImpl.grantUsers` — the same table, the same statements, from both directions. Splitting them puts a two-writer table across a network boundary for no measured benefit. |
| Split icons from profiles                                | `ProfileRepositoryImpl.getProfile` reads `profile_icons` on every profile hydration. Splitting adds a synchronous hop to the most common profile read to isolate one `bytea` column capped at 2 MB by `spring.servlet.multipart.max-file-size`.                  |

### 6.2 The two cross-boundary reads

Exactly two reads cross the line. Neither needs a synchronous call between the
services.

**Read 1 — identity needs `profile_id` when it mints a token.**
`JwtService.generateToken` writes a `profile_id` claim, sourced from
`SecurityUser.getProfileId()`, which reaches into the embedded `Profile`.
Downstream, eleven of the fifteen profile endpoints authorize with
`#profileId == authentication.principal.getProfileId()` — so the claim is not
incidental, it is what the profile authorization model runs on.

_Resolution:_ `identity-service` reads `auth.profiles.profile_id` directly at
token-mint time, one indexed lookup by `user_id` once per login, and stamps it
into the claim. `profile-service` then reads `profile_id` from the verified
token and never calls `identity-service` on the request path. This is a
read-only cross-context table read on a shared schema, which the descoped plan
explicitly permits, and it happens once per login rather than once per request.
It requires the index on `auth.profiles.user_id` from section 3.7.

**Read 2 — profile needs to know a user exists, and who is acting.**
`ProfileService.createProfile` validates its `userId` argument against
`UserRepository.findById` before inserting, and `getUserIdByEmailAddress`
resolves the acting principal for the `created_by_user_id` and
`modified_by_user_id` audit columns on every profile write.

_Resolution:_ the acting user id is already in the token as the `user_id` claim,
so the audit-column lookup disappears entirely — that is pure per-request work
the JWT already answers. The existence check becomes a foreign-key constraint on
`auth.profiles.user_id` enforced by Postgres, which is stronger than the current
application-level check and costs nothing on the request path.

**Severing the embedded profile.** The `User` payload today embeds the whole
profile aggregate (section 5.2), which would force `identity-service` to call
`profile-service` on every user read — the one genuinely bad outcome available
here. **Amendment to the contract:** the `identity-service` user representation
drops the embedded profile and exposes `profile_id` only. Clients that need the
profile fetch `GET /api/profiles/user/{userId}` from `profile-service`.

This is the amendment the audit produces, and it pays for itself: it removes 3
of the 5 statements per row measured in 4.2, taking a default-page
`GET /api/users` from roughly 501 statements to 201, before any batching.

### 6.3 Measured baseline

All figures from Prometheus for the prd pod
`jdwlabs-usersrole-prd-65c86d47df-8vcl5`, window 7 d ending 2026-09-05, unless
stated. Binary units: 1 MiB = 1,048,576 B.

**Memory — the JVM process**

| Metric                                | Bytes       | MiB       |
| ------------------------------------- | ----------- | --------- |
| Working set, 7 d minimum (idle floor) | 220,700,672 | **210.5** |
| Working set, 7 d mean (loaded)        | 274,771,109 | **262.0** |
| Working set, 7 d peak                 | 293,842,944 | **280.2** |
| Container RSS, 7 d minimum            | 218,480,640 | 208.4     |
| Container RSS, 7 d peak               | 290,033,664 | **276.6** |

**Memory — inside the JVM**

| Metric                     | Bytes       | MiB   |
| -------------------------- | ----------- | ----- |
| Heap used, 7 d peak        | 45,510,024  | 43.4  |
| Non-heap used, 7 d peak    | 136,450,488 | 130.1 |
| Total committed, 7 d peak  | 201,519,104 | 192.2 |
| GC live data set, 7 d peak | 25,633,360  | 24.4  |
| Metaspace, instantaneous   | 75,687,304  | 72.2  |
| Heap used, instantaneous   | 28,682,472  | 27.4  |

The live object graph this service actually maintains is **24.4 MiB**. The
container holds **276.6 MiB** of RSS to do it. Non-heap exceeds heap by 3x, and
Metaspace alone — pure JVM class metadata, carrying no application state — is
**72.2 MiB**, itself larger than the entire measured footprint of the existing
Go service on this platform. That ratio, not the heap number, is the case for
the rewrite.

**CPU**

| Metric              | Cores    | Millicores |
| ------------------- | -------- | ---------- |
| 7 d mean            | 0.000832 | **0.83**   |
| 7 d peak (5 m rate) | 0.009278 | **9.28**   |
| Configured request  | 0.5      | 500        |
| Configured limit    | 1.0      | 1000       |

The CPU request is **~600x** the measured mean and **~54x** the measured peak.

**Connection pool**

| Metric                       | Measured | Configured |
| ---------------------------- | -------- | ---------- |
| Active connections, 7 d peak | **0**    | —          |
| Active connections, 7 d mean | 0        | —          |
| Total connections, 7 d peak  | 5        | —          |
| Idle connections, 7 d peak   | 5        | —          |
| Pending threads, 7 d peak    | **0**    | —          |
| `maximum-pool-size`          | —        | 10         |
| `minimum-idle`               | —        | 2          |
| Acquire time, 7 d peak       | 38.79 ms | —          |

Active connections never rose above zero across 20,160 scrapes, and nothing ever
queued for a connection. The pool oscillates between 2 and 5 connections purely
from `max-lifetime: 300000` recycling idle ones every five minutes. The
configured maximum of 10 is not supported by any measurement.

**Runtime**

| Metric                   | Value   |
| ------------------------ | ------- |
| Application startup time | 6.246 s |
| Live threads, 7 d peak   | 27      |
| Peak threads, 7 d        | 27      |

**Latency, where samples exist**

| URI                    | Method | 7 d p50  | 7 d p95  | 7 d p99  |
| ---------------------- | ------ | -------- | -------- | -------- |
| `/actuator/health/**`  | GET    | 0.509 ms | 0.968 ms | 1.398 ms |
| `/actuator/prometheus` | GET    | 2.328 ms | 3.605 ms | 4.265 ms |
| `/**` (404)            | GET    | 25.17 ms | 27.68 ms | 27.91 ms |
| `UNKNOWN` (401)        | GET    | 0.503 ms | 0.956 ms | 0.996 ms |
| `UNKNOWN` (401)        | POST   | 0.500 ms | 0.950 ms | 0.990 ms |

These prove the histogram pipeline works end to end. No business URI has a
7-day percentile, because no business URI has a 7-day sample.

The only business-endpoint timings that exist anywhere are single observations
over 30 days, which are means of n=1 and are **not** percentiles:

| URI                   | Method | 30 d mean | n   |
| --------------------- | ------ | --------- | --- |
| `/auth/authenticate`  | POST   | 87.96 ms  | 1   |
| `/api/users/{userId}` | GET    | 35.87 ms  | 1   |

From this audit's own non-prod runs, `POST /auth/authenticate` returning 401
averaged **48.20 ms** over n=4. That figure is dominated by bcrypt verification;
the 5-statement fan-out against 9-row tables contributes well under a
millisecond, so it should be read as the cost of the password hash and not of
the data access.

**Reference point: the Go service already running on this platform**

Measured over the same 7-day window, `servicediscovery` in prd:

| Metric                  | Value                                          |
| ----------------------- | ---------------------------------------------- |
| Working set, 7 d mean   | 4.24 – 5.59 MB (2 of 3 pods); 27.70 MB (third) |
| Container RSS, 7 d peak | 3.32 / 4.60 / 5.41 MB                          |
| CPU, 7 d peak           | 0.14 – 0.70 millicores                         |
| Configured request      | 32 Mi / 10 m                                   |
| Configured limit        | 64 Mi / 50 m                                   |

**Restating the epic's headline metric against measured values.** The
decomposition analysis framed the gap as "1 Gi per replica versus tens of MiB",
and the production chart comment records live steady state at "~216 Mi". Both
are wrong in the same direction, and neither is the number to plan against:

- The chart no longer reserves 1 Gi. It requests **512 Mi** with a 2 Gi limit.
- "~216 Mi" is close to the measured 7-day **floor** (210.5 MiB), not the steady
  state. The measured mean is **262.0 MiB** and the peak is **280.2 MiB**.
- The real comparison is **274,771,109 B measured JVM working set against
  5,410,816 B measured Go RSS peak** on this same cluster — approximately
  **51x**. The "1 Gi vs tens of MiB" framing understates the ratio while
  overstating the JVM's actual reservation; "216 Mi" understates both.

One operational note falls out of this. The 512 Mi request is 1.83x the measured
280.2 MiB peak, not the "roughly 2x steady-state" the chart comment claims,
because the comment is anchored to the floor. The margin is real but thinner
than documented.

### 6.4 Resource sizing for the two Go services

Derived from the measurements above. The anchor is the measured Go service on
this cluster (5.41 MB RSS peak, 0.70 millicore peak), adjusted for what each new
service does that the anchor does not.

| Service            | Request memory | Request CPU | Limit memory | Limit CPU | Reasoning                                                                                                                                                                                                                        |
| ------------------ | -------------- | ----------- | ------------ | --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `identity-service` | **32 Mi**      | **10 m**    | **128 Mi**   | **200 m** | Matches the measured Go sibling's request, which peaks at 5.41 MB RSS. Adds a connection pool and bcrypt. The limit gives 4x headroom over the sibling's own limit for bcrypt's per-verification working set.                    |
| `profile-service`  | **48 Mi**      | **10 m**    | **192 Mi**   | **500 m** | Same base, plus icon handling. `max-file-size` is 2 MB and icons are read whole into memory; at 10 concurrent requests that is up to 20 MB transient, plus response buffering. The higher CPU limit absorbs image upload bursts. |

Both figures are far above the measured 0.83 millicore mean for the whole
monolith, deliberately: these are floors for a service that must survive a
restart storm and a probe schedule, not a fit to a workload of two requests per
month.

**Pool sizes, derived rather than guessed.** The monolith's configured maximum
of 10 was never approached: 0 active connections at peak, 0 pending, 5 total at
peak. Splitting into two services should not increase total connection pressure
on the shared cluster.

| Service            | `max_open_conns` | `max_idle_conns` | Basis                                                                                                                                                             |
| ------------------ | ---------------- | ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `identity-service` | **5**            | **2**            | Half the monolith's configured maximum, keeping the total at 10. Still ~5x above the measured peak total of 5 and infinitely above the measured active peak of 0. |
| `profile-service`  | **5**            | **2**            | As above.                                                                                                                                                         |

Pool size is driven by request concurrency, not by statements per request: the
fan-out in section 4 runs sequentially on one connection. Measured concurrency
is zero, so 5 per service is already generous.

## 7. PromQL used

Re-runnable as-is against the in-cluster Prometheus. Substitute the namespace to
compare environments.

```promql
# Working set: 7-day floor, mean and peak
min_over_time(container_memory_working_set_bytes{namespace="jdwlabs-prd",pod=~"jdwlabs-usersrole-.*",container="usersrole"}[7d])
avg_over_time(container_memory_working_set_bytes{namespace="jdwlabs-prd",pod=~"jdwlabs-usersrole-.*",container="usersrole"}[7d])
max_over_time(container_memory_working_set_bytes{namespace="jdwlabs-prd",pod=~"jdwlabs-usersrole-.*",container="usersrole"}[7d])

# Container RSS: 7-day floor and peak
min_over_time(container_memory_rss{namespace="jdwlabs-prd",pod=~"jdwlabs-usersrole-.*",container="usersrole"}[7d])
max_over_time(container_memory_rss{namespace="jdwlabs-prd",pod=~"jdwlabs-usersrole-.*",container="usersrole"}[7d])

# CPU: 7-day mean and peak, in cores
avg_over_time(rate(container_cpu_usage_seconds_total{namespace="jdwlabs-prd",pod=~"jdwlabs-usersrole-.*",container="usersrole"}[5m])[7d:5m])
max_over_time(rate(container_cpu_usage_seconds_total{namespace="jdwlabs-prd",pod=~"jdwlabs-usersrole-.*",container="usersrole"}[5m])[7d:5m])

# JVM memory
max_over_time(sum(jvm_memory_used_bytes{namespace="jdwlabs-prd",area="heap"})[7d:1m])
max_over_time(sum(jvm_memory_used_bytes{namespace="jdwlabs-prd",area="nonheap"})[7d:1m])
max_over_time(sum(jvm_memory_committed_bytes{namespace="jdwlabs-prd"})[7d:1m])
max_over_time(jvm_gc_live_data_size_bytes{namespace="jdwlabs-prd"}[7d])
sum by (area) (jvm_memory_used_bytes{namespace="jdwlabs-prd",job=~".*usersrole.*"})
jvm_memory_used_bytes{namespace="jdwlabs-prd"}                    # per-pool, incl. Metaspace

# Threads and startup
max_over_time(jvm_threads_live_threads{namespace="jdwlabs-prd"}[7d])
max_over_time(jvm_threads_peak_threads{namespace="jdwlabs-prd"}[7d])
max_over_time(application_started_time_seconds{namespace="jdwlabs-prd"}[7d])

# HikariCP against the configured maximum
max_over_time(hikaricp_connections_active{namespace="jdwlabs-prd"}[7d])
avg_over_time(hikaricp_connections_active{namespace="jdwlabs-prd"}[7d])
max_over_time(hikaricp_connections{namespace="jdwlabs-prd"}[7d])
max_over_time(hikaricp_connections_idle{namespace="jdwlabs-prd"}[7d])
max_over_time(hikaricp_connections_pending{namespace="jdwlabs-prd"}[7d])
max_over_time(hikaricp_connections_acquire_seconds_max{namespace="jdwlabs-prd"}[7d])
hikaricp_connections_max{namespace="jdwlabs-prd"}
hikaricp_connections_min{namespace="jdwlabs-prd"}

# Traffic mix and share
sum by (uri,method,status) (increase(http_server_requests_seconds_count{namespace="jdwlabs-prd"}[7d]))
sum by (uri,method,status) (increase(http_server_requests_seconds_count{namespace="jdwlabs-prd",uri!~"/actuator.*"}[30d]))

# Latency percentiles per URI
histogram_quantile(0.50, sum by (uri,method,le) (rate(http_server_requests_seconds_bucket{namespace="jdwlabs-prd"}[7d])))
histogram_quantile(0.95, sum by (uri,method,le) (rate(http_server_requests_seconds_bucket{namespace="jdwlabs-prd"}[7d])))
histogram_quantile(0.99, sum by (uri,method,le) (rate(http_server_requests_seconds_bucket{namespace="jdwlabs-prd"}[7d])))

# Mean latency where percentiles have no samples (state n alongside)
  sum by (uri,method) (increase(http_server_requests_seconds_sum{namespace="jdwlabs-prd"}[30d]))
/ sum by (uri,method) (increase(http_server_requests_seconds_count{namespace="jdwlabs-prd"}[30d]))

# Go reference point
avg_over_time(container_memory_working_set_bytes{namespace=~"jdwlabs-(prd|non)",pod=~"jdwlabs-servicediscovery-.*",container="servicediscovery"}[7d])
max_over_time(container_memory_rss{namespace=~"jdwlabs-(prd|non)",pod=~"jdwlabs-servicediscovery-.*",container="servicediscovery"}[7d])
max_over_time(rate(container_cpu_usage_seconds_total{namespace="jdwlabs-prd",pod=~"jdwlabs-servicediscovery-.*",container="servicediscovery"}[5m])[7d:5m])
```

Fan-out snapshot query, run against the non cluster before and after a single
request:

```sql
SELECT relname, seq_scan, idx_scan, seq_scan + idx_scan
FROM pg_stat_user_tables
WHERE schemaname = 'auth'
ORDER BY relname;
```

## 8. Open risks

| #   | Risk                                                                                                                                                                                                                | Impact | Mitigation                                                                                                                                                      |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **No production load exists to validate against.** Two business requests in 30 days. Cutover cannot be verified by comparing production latency before and after, because there is no before.                       | High   | Verify with a synthetic benchmark against a seeded dataset, run identically against the monolith and both Go services. Define the dataset in the contract work. |
| 2   | **No latency SLO can be derived from history.** Section 6.3 has no business-endpoint percentiles.                                                                                                                   | High   | Set targets from the synthetic benchmark. Do not carry the n=1 means in 6.3 into any SLO.                                                                       |
| 3   | **The bcrypt password hash is returned by every `User` response**, including the public `POST /auth/user`.                                                                                                          | High   | The Go contract defines an explicit user response type that omits the credential. Treat as a defect to close during the rewrite, not a behaviour to preserve.   |
| 4   | **Four foreign-key columns on the hydration path are unindexed** (section 3.7). Invisible at 9 rows, quadratic as data grows.                                                                                       | High   | Add the indexes alongside the Go data-access work. Fixing the N+1 without them still leaves `WHERE role_id = ?` scanning a growing join table.                  |
| 5   | **All fan-out measurements come from a 3-user fixture.** The per-row cost is exact and row-count-independent, but the per-statement _latency_ is not — every table fits one page today.                             | Medium | Re-run section 4's method against a seeded dataset before fixing the pool sizes in 6.4.                                                                         |
| 6   | **`GET /api/roles` is unpaginated** and its fan-out is `1 + N` over the whole table.                                                                                                                                | Medium | Paginate it in the Go contract. It is a behaviour change, so it needs recording in the contract rather than slipping in.                                        |
| 7   | **Any authenticated principal can enumerate all roles** (section 2.3). Widening or narrowing this during the rewrite is a security-relevant behaviour change either way.                                            | Medium | Decide explicitly in the contract work and record the decision.                                                                                                 |
| 8   | **`pg_stat_statements` is unavailable**, so per-statement timing was never observed — only statement counts.                                                                                                        | Medium | Load the extension (section 1.1) before the benchmark, so the cutover check has per-statement timings rather than counts.                                       |
| 9   | **The icon DAO's identifier is inconsistent** between `create`, `findById`, `update` and `deleteById` (section 3.6).                                                                                                | Medium | Pick one identifier for the icon resource in the contract. Do not port the inconsistency.                                                                       |
| 10  | **The `profile_id` claim is minted at login and never refreshed.** A profile created after a token is issued leaves that token with a null claim until it expires — 2 hours at the configured `expiration-time-ms`. | Medium | Either mint a fresh token on profile creation, or have `profile-service` fall back to a `user_id` lookup when the claim is absent. Decide in the contract work. |
| 11  | **Deleting a user cascades across the boundary** — `UserRepositoryImpl.deleteById` writes to `addresses`, `profile_icons` and `profiles`, all owned by the other service.                                           | Medium | Resolve with `ON DELETE CASCADE` in the schema rather than a cross-service call, since both services share the schema.                                          |
| 12  | **The production memory request has less headroom than its own comment claims** — 512 Mi is 1.83x the measured 280.2 MiB peak, not 2x steady state.                                                                 | Low    | No action while the JVM is being retired. Note it if the cutover slips.                                                                                         |
