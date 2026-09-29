// Generated from the frozen usersrole contracts. Regenerate with
// `pnpm exec nx run platform-live-e2e:generate-contract-types`; never edit by hand.

export interface paths {
    "/auth/authenticate": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Exchange credentials for a JWT. */
        post: operations["authenticate"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/user": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Public self-registration. */
        post: operations["registerUser"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/users": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List users, one page at a time. */
        get: operations["getAllUsers"];
        put?: never;
        /** Create a user as an authenticated caller. */
        post: operations["createUser"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/users/{userId}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Read one user. */
        get: operations["getUserById"];
        /**
         * Replace a user's email address and password.
         * @description The request body is the same `UserRequestDTO` as registration, and both
         *     fields are required, so every update rewrites the bcrypt hash. A client
         *     that wants to change only the email address must resend the password.
         *     Frozen as-is.
         *
         *     Updating to an email address another user already holds violates
         *     `users_email_address_key` and surfaces as 500, not 409 — the unique
         *     violation is raised by Postgres and no handler maps it.
         */
        put: operations["updateUser"];
        post?: never;
        /**
         * Delete a user and everything hanging off it.
         * @description Deleting a user removes its role grants, its profile, and that profile's
         *     addresses and icon. Those last three tables belong to `profile-service`,
         *     so this is the one write that crosses the service boundary.
         *
         *     Frozen as one local transaction in `identity-service`, not a call into
         *     `profile-service`: both services share the `auth` schema, so the cascade
         *     stays a write `identity-service` performs itself. See `x-delete-cascade`.
         *
         *     Deleting an id that does not exist is a no-op and still returns 204 — the
         *     handler issues the deletes without checking for the row first.
         */
        delete: operations["deleteUser"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/users/email/{emailAddress}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Read one user by email address. */
        get: operations["getUserByEmailAddress"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/users/{userId}/roles/grant": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /** Grant roles to a user. */
        put: operations["grantRolesToUser"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/users/{userId}/roles/revoke": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /** Revoke roles from a user. */
        put: operations["revokeRolesFromUser"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/roles": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List roles, one page at a time.
         * @description Paginated in this contract; unpaginated today. See `x-behaviour-change`.
         */
        get: operations["getAllRoles"];
        put?: never;
        /** Create a role. */
        post: operations["createRole"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/roles/{roleId}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Read one role. */
        get: operations["getRoleById"];
        /**
         * Replace a role's name and description.
         * @description `auth.roles.role_name` is `NOT NULL UNIQUE`, and the update has no
         *     pre-check for a name already in use — unlike the create, which does look
         *     the name up first and answers 409. Renaming a role onto a name another
         *     role holds therefore reaches Postgres as a unique violation that no
         *     handler maps, and surfaces as 500. Frozen: the asymmetry with the create
         *     is real, and clients key their messages off the status code.
         */
        put: operations["updateRole"];
        post?: never;
        /**
         * Delete a role.
         * @description A no-op for an id that does not exist, and still 204 — the handler
         *     deletes without reading first.
         */
        delete: operations["deleteRole"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/roles/name/{roleName}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Read one role by name. */
        get: operations["getRoleByName"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/roles/{roleId}/users/grant": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /**
         * Grant a role to users.
         * @description The same write as `PUT /api/users/{userId}/roles/grant` from the other
         *     direction — both insert into `auth.users_roles`. That two-writer table is
         *     why users and roles stay in one service.
         */
        put: operations["grantUsersToRole"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/roles/{roleId}/users/revoke": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /** Revoke a role from users. */
        put: operations["revokeUsersFromRole"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        /** @description The registration, creation and update body. Both fields are required on all three. */
        UserRequestDTO: {
            /**
             * Format: email
             * @description The pattern below is necessary but not sufficient. `@Email` applies
             *     Hibernate Validator's own address checks first and this pattern
             *     second, so an address must satisfy both: the local part is at most 64
             *     characters of dot-separated non-empty atoms, the domain is at most
             *     255 characters of dot-separated labels of 1 to 63 characters, and no
             *     domain label begins or ends with a hyphen. The pattern is what
             *     narrows the character set — word characters, dots and hyphens either
             *     side of the `@`, and a dot-separated alphabetic TLD of at least two
             *     characters — which is looser than RFC 5322 in the characters it
             *     allows and stricter in the shapes.
             */
            emailAddress: string;
            /**
             * @description Accepted on input, never returned. At least 8 characters with an
             *     upper-case letter, a lower-case letter, a digit and a symbol. Stored
             *     bcrypt-encoded.
             */
            password: string;
        };
        AuthResponseDTO: {
            jwtToken: string;
        };
        /**
         * @description The user representation. Carries no password field in any response — see
         *     `x-password-contract` — and no embedded profile — see
         *     `x-behaviour-change`.
         */
        User: {
            /** Format: int64 */
            id?: number;
            /** Format: email */
            emailAddress?: string;
            /** @description `ACTIVE` for every row this service creates. */
            status?: string;
            roles?: components["schemas"]["UserRole"][];
            /**
             * Format: int64
             * @description The user's profile id, or null when it has no profile row. Resolved
             *     by `identity-service` from `auth.profiles.user_id`, a read-only
             *     cross-context read on the shared schema. The same lookup supplies the
             *     `profile_id` token claim.
             */
            profileId?: number | null;
            /** Format: int64 */
            createdByUserId?: number;
            /** Format: date-time */
            createdTime?: string;
            /** Format: int64 */
            modifiedByUserId?: number;
            /** Format: date-time */
            modifiedTime?: string;
        };
        /** @description One row of `auth.users_roles`. Written from both the user and the role side. */
        UserRole: {
            /** Format: int64 */
            userId?: number;
            /** Format: int64 */
            roleId?: number;
            /** Format: int64 */
            createdByUserId?: number;
            /** Format: date-time */
            createdTime?: string;
        };
        RoleRequestDTO: {
            name: string;
            description: string;
        };
        Role: {
            /** Format: int64 */
            id?: number;
            name?: string;
            description?: string;
            status?: string;
            /**
             * @description The role's grants. Visible to any authenticated caller — see the
             *     `x-authorization-decision` on `GET /api/roles`.
             */
            users?: components["schemas"]["UserRole"][];
            /** Format: int64 */
            createdByUserId?: number;
            /** Format: date-time */
            createdTime?: string;
            /** Format: int64 */
            modifiedByUserId?: number;
            /** Format: date-time */
            modifiedTime?: string;
        };
        RoleIdList: number[];
        UserIdList: number[];
        /** @description Field name to validation message, one entry per rejected field. */
        ValidationErrors: {
            [key: string]: string;
        };
        /**
         * @description The servlet container's error body, produced whenever a status is set
         *     through `sendError` rather than composed by a handler. Its exact keys are
         *     a framework detail; no client reads them, and the frontends' shared error
         *     helper switches on the status code alone.
         */
        ContainerError: {
            /** Format: date-time */
            timestamp?: string;
            /** Format: int32 */
            status?: number;
            error?: string;
            message?: string;
            path?: string;
        };
    };
    responses: {
        /**
         * @description A path variable or a `page`/`size` query parameter was not convertible to
         *     the numeric type the handler declares. Spring's argument resolution fails
         *     before the handler runs and nothing in `GlobalExceptionHandler` catches
         *     the resulting exception, so this is a status the container sets through
         *     `sendError` rather than one a handler composed — and it carries the body
         *     the container renders, for the reason `x-container-error` sets out.
         *     Distinct from `BadRequest`, which is what the two body handlers produce;
         *     no client reads any of the three, and `http-error-message.util.ts` keys
         *     on the status alone.
         */
        UnconvertableParameter: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ContainerError"];
            };
        };
        /**
         * @description Three body shapes share this status on the operations that reference it.
         *     Bean-validation failures produce a JSON object of field name to message,
         *     from `GlobalExceptionHandler.handle(MethodArgumentNotValidException)`.
         *     An unparseable body produces the fixed `text/plain` string
         *     `Request body is invalid. Please check the format and try again.` And a
         *     numeric path variable or paging parameter that will not convert produces
         *     the container's own `ContainerError` — the same 400 that
         *     `UnconvertableParameter` describes, merged in here because an operation
         *     answers one status with one response object.
         */
        BadRequest: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ValidationErrors"] | components["schemas"]["ContainerError"];
                "text/plain": string;
            };
        };
        /**
         * @description No token, an expired token, or a token that fails verification. Written
         *     by the authentication entry point through `HttpServletResponse.sendError`.
         *     `server.error.include-message` is unset, so Boot's default of `never`
         *     applies and no error page renders: the response carries no body and no
         *     `Content-Type`, for every `Accept` value. Clients key off the status and
         *     the header; there is no body to key off.
         *
         *     A second route reaches this response, and it replaces a status rather
         *     than raising one: any status the container sets on a request carrying no
         *     token. `/error` is outside the `permitAll` matchers, so the forward
         *     `sendError` makes is refused a second time and the entry point answers
         *     instead — see `x-container-error`. That is why `POST /auth/user` lists a
         *     401 although it takes no token, and why an unauthenticated request to an
         *     unmapped path under `/auth/**` answers 401 rather than 404.
         */
        Unauthorized: {
            headers: {
                /** @description `Authentication Required` */
                "Access-Denied-Reason": string;
                [name: string]: unknown;
            };
            content?: never;
        };
        /**
         * @description Authenticated but not permitted: the operation's `x-authorization` rule
         *     rejected the principal, or the elevated-role guard on the grant and
         *     revoke operations did. Written by the access-denied handler through
         *     `sendError`, same as `Unauthorized` above — but unlike it, this one is
         *     not empty. The caller here already has a verified token, so the
         *     internal forward to `/error` that `sendError` triggers re-authenticates
         *     with that same token and reaches `BasicErrorController`, which renders
         *     Boot's standard error body. `message` is absent because
         *     `server.error.include-message` is `never`, not because the body itself
         *     is empty.
         */
        Forbidden: {
            headers: {
                /** @description `Not Authorized` */
                "Access-Denied-Reason": string;
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ContainerError"];
            };
        };
        /**
         * @description `ResourceNotFoundException`. The body is the exception message as
         *     `text/plain`, e.g. `User not found with id 42`.
         */
        NotFound: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "text/plain": string;
            };
        };
        /**
         * @description `ResourceExistsException`. The body is the exception message as
         *     `text/plain`, e.g. `User already exists with email address a@b.c`.
         */
        Conflict: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "text/plain": string;
            };
        };
    };
    parameters: {
        /**
         * @description Zero-based page index. Out-of-range input is clamped, not rejected: a
         *     negative page is served as page 0 with a 200, never a 400.
         */
        Page: number;
        /**
         * @description Page size, bounded server-side to at most 500 rows so the listing always
         *     issues a bounded query whatever the caller asks for. Like `page`, out-of
         *     -range input is clamped rather than rejected: `size=0` is served as 1 and
         *     `size=100000` as 500, both with a 200. `minimum` and `maximum` here
         *     describe the range the service will actually serve, not a range it
         *     validates — a request outside it succeeds.
         */
        Size: number;
        UserId: number;
        RoleId: number;
        /**
         * @description Declared explicitly because the handler reads the raw header to resolve
         *     the acting user for the audit columns. Redundant with the `bearerAuth`
         *     security scheme and kept only because removing it would change the served
         *     document. In the Go services the acting user comes from the verified
         *     `user_id` claim and this parameter disappears; it is frozen here because
         *     it is part of today's document.
         */
        AuthorizationHeader: string;
    };
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    authenticate: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["UserRequestDTO"];
            };
        };
        responses: {
            /** @description Token minted. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AuthResponseDTO"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    registerUser: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["UserRequestDTO"];
            };
        };
        responses: {
            /** @description User created. springdoc reports 200; the handler builds `HttpStatus.CREATED`. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            409: components["responses"]["Conflict"];
        };
    };
    getAllUsers: {
        parameters: {
            query?: {
                /**
                 * @description Zero-based page index. Out-of-range input is clamped, not rejected: a
                 *     negative page is served as page 0 with a 200, never a 400.
                 */
                page?: components["parameters"]["Page"];
                /**
                 * @description Page size, bounded server-side to at most 500 rows so the listing always
                 *     issues a bounded query whatever the caller asks for. Like `page`, out-of
                 *     -range input is clamped rather than rejected: `size=0` is served as 1 and
                 *     `size=100000` as 500, both with a 200. `minimum` and `maximum` here
                 *     describe the range the service will actually serve, not a range it
                 *     validates — a request outside it succeeds.
                 */
                size?: components["parameters"]["Size"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description A page of users as a bare array. There is no envelope and no total count. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"][];
                };
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    createUser: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["UserRequestDTO"];
            };
        };
        responses: {
            /** @description User created. springdoc reports 200; the handler builds `HttpStatus.CREATED`. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            409: components["responses"]["Conflict"];
        };
    };
    getUserById: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The user. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    updateUser: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["UserRequestDTO"];
            };
        };
        responses: {
            /** @description The updated user. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            /**
             * @description The new email address is already taken. A unique-constraint violation
             *     with no handler mapping it. Frozen because clients key their messages
             *     off the status code and changing it is a visible behaviour change.
             */
            500: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ContainerError"];
                };
            };
        };
    };
    deleteUser: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /**
             * @description Deleted, no body. springdoc reports `200` with a `User` body because
             *     the handler's declared return type is `ResponseEntity<User>`; the
             *     body it actually builds is `noContent()`.
             */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    getUserByEmailAddress: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                emailAddress: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The user. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    grantRolesToUser: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["RoleIdList"];
            };
        };
        responses: {
            /** @description The user with its updated grants. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    revokeRolesFromUser: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["RoleIdList"];
            };
        };
        responses: {
            /** @description The user with its updated grants. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    getAllRoles: {
        parameters: {
            query?: {
                /**
                 * @description Zero-based page index. Out-of-range input is clamped, not rejected: a
                 *     negative page is served as page 0 with a 200, never a 400.
                 */
                page?: components["parameters"]["Page"];
                /**
                 * @description Page size, bounded server-side to at most 500 rows so the listing always
                 *     issues a bounded query whatever the caller asks for. Like `page`, out-of
                 *     -range input is clamped rather than rejected: `size=0` is served as 1 and
                 *     `size=100000` as 500, both with a 200. `minimum` and `maximum` here
                 *     describe the range the service will actually serve, not a range it
                 *     validates — a request outside it succeeds.
                 */
                size?: components["parameters"]["Size"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description A page of roles as a bare array. No envelope, no total count. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Role"][];
                };
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
        };
    };
    createRole: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["RoleRequestDTO"];
            };
        };
        responses: {
            /** @description Role created. springdoc reports 200; the handler builds `HttpStatus.CREATED`. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Role"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            409: components["responses"]["Conflict"];
        };
    };
    getRoleById: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                roleId: components["parameters"]["RoleId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The role. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Role"];
                };
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    updateRole: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                roleId: components["parameters"]["RoleId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["RoleRequestDTO"];
            };
        };
        responses: {
            /** @description The updated role. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Role"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            /**
             * @description The new name is already held by another role. A unique-constraint
             *     violation with no handler mapping it, so the body is the container's
             *     error representation rather than the 409 the create would give.
             */
            500: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ContainerError"];
                };
            };
        };
    };
    deleteRole: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                roleId: components["parameters"]["RoleId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Deleted, no body. springdoc reports 200; the handler builds `noContent()`. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    getRoleByName: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                roleName: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The role. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Role"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    grantUsersToRole: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                roleId: components["parameters"]["RoleId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["UserIdList"];
            };
        };
        responses: {
            /** @description The role with its updated grants. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Role"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    revokeUsersFromRole: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go services the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears; it is frozen here because
                 *     it is part of today's document.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                roleId: components["parameters"]["RoleId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["UserIdList"];
            };
        };
        responses: {
            /** @description The role with its updated grants. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Role"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
}
