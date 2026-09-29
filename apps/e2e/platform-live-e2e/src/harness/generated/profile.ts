// Generated from the frozen usersrole contracts. Regenerate with
// `pnpm exec nx run platform-live-e2e:generate-contract-types`; never edit by hand.

export interface paths {
    "/api/profiles": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List profiles, one page at a time. */
        get: operations["getProfiles"];
        put?: never;
        /**
         * Create a profile for a user.
         * @description One profile per user. A second create for the same user id is 409.
         */
        post: operations["createProfile"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/profiles/{profileId}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Read one profile. */
        get: operations["getProfileById"];
        /**
         * Replace a profile's name fields and birthdate.
         * @description `userId` is not in the update body: a profile cannot be moved between users.
         */
        put: operations["updateProfileById"];
        post?: never;
        /**
         * Delete a profile with its addresses and icon.
         * @description A no-op for an id that does not exist, and still 204.
         *
         *     Deleting your own profile changes what your existing token can do, and
         *     the split changes it differently — see `x-behaviour-change`.
         */
        delete: operations["deleteProfileById"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/profiles/by-user/{userId}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read a user's profile.
         * @description The operation `identity-service` clients use once the `User` payload no
         *     longer embeds the profile aggregate.
         */
        get: operations["getProfileByUserId"];
        /** Replace a user's profile. */
        put: operations["updateProfileByUserId"];
        post?: never;
        /**
         * Delete a user's profile with its addresses and icon.
         * @description A no-op for a user id with no profile, and still 204.
         */
        delete: operations["deleteProfileByUserId"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/profiles/{profileId}/address": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Add an address to a profile.
         * @description Returns 200, not 201, unlike every other create in either service. The
         *     handler builds `ResponseEntity.ok`. Frozen as-is: the asymmetry is
         *     harmless and changing it would be a client-visible change for nothing.
         *
         *     The response is the whole profile, not the created address, so a client
         *     that needs the new address id reads it from `addresses`.
         */
        post: operations["addAddress"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/profiles/{profileId}/address/{addressId}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /**
         * Replace one of a profile's addresses.
         * @description Scoped to the profile: the handler resolves the address inside that
         *     profile's own address set, so an `addressId` belonging to another profile
         *     is 404 rather than an edit.
         */
        put: operations["updateAddress"];
        post?: never;
        /**
         * Delete one of a profile's addresses.
         * @description Scoped to the profile in the path, and 404 when the address does not
         *     belong to it. This is a transcription, not a deviation: the scoping is in
         *     `AddressDaoPostgres.deleteByProfileIdAndAddressId`, whose statement is
         *     `DELETE FROM auth.addresses WHERE address_id = :addressId AND profile_id
         *     = :profileId`, and `ProfileService.deleteAddress` raises
         *     `ResourceNotFoundException` when it deletes no row.
         *
         *     It was not always so, and the history is worth keeping: until recently
         *     the handler took `profileId`, used it for the authorization check, and
         *     then deleted by `addressId` alone, so any authenticated principal with a
         *     profile could delete any address in the table by guessing a sequential id
         *     and get 204 for it. This contract specified the scoped behaviour while
         *     that was still true, because writing an insecure direct object reference
         *     into the document the Go services are built and parity-tested against
         *     would have had the generated suite certify it. The application has since
         *     been fixed independently, so contract and code now agree.
         */
        delete: operations["deleteAddress"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/profiles/{profileId}/icon": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Download a profile's icon.
         * @description The only non-JSON response in either service: the raw bytes of the stored
         *     icon, declared `image/png` by the handler's `produces`. The bytes are
         *     whatever was uploaded — nothing validates that they are a PNG, so the
         *     declared type is a promise the service does not check. Frozen; a Go
         *     implementation that starts sniffing content types would change behaviour
         *     for existing rows.
         */
        get: operations["getProfileIcon"];
        /**
         * Replace a profile's icon.
         * @description `multipart/form-data` with one part named `icon`, as for the upload.
         *
         *     Replacing an icon on a profile that has none is 500, not 404: the handler
         *     reads the current icon's id to carry it onto the replacement and
         *     dereferences null when there is no current icon. Frozen as a documented
         *     status rather than silently corrected, because a client that gets 500
         *     today would get 404 tomorrow, and the frontends' error text is keyed on
         *     the status. Correcting it is a behaviour change and belongs with the
         *     service rewrite, not with the freeze.
         */
        put: operations["updateIcon"];
        /**
         * Upload a profile's first icon.
         * @description `multipart/form-data` with one part named `icon`, capped by
         *     `x-multipart-limits`. springdoc declares the body `application/json`
         *     because it renders the `MultipartFile` parameter as a JSON object with a
         *     binary property; the handler consumes multipart and nothing else.
         */
        post: operations["addIcon"];
        /**
         * Delete a profile's icon.
         * @description Keyed on `profile_id` — see `x-icon-identifier`. A no-op for a profile
         *     with no icon, and still 204.
         */
        delete: operations["deleteIcon"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        ProfileCreateRequestDTO: {
            firstName: string;
            middleName?: string;
            lastName: string;
            /** Format: date-time */
            birthdate: string;
            /**
             * Format: int64
             * @description The user this profile belongs to. Immutable once created.
             */
            userId: number;
        };
        /** @description The create body without `userId`, which cannot be changed. */
        ProfileUpdateRequestDTO: {
            firstName: string;
            middleName?: string;
            lastName: string;
            /** Format: date-time */
            birthdate: string;
        };
        AddressRequestDTO: {
            addressLine1: string;
            addressLine2?: string;
            city: string;
            stateProvince: string;
            postalCode: string;
            country: string;
        };
        /**
         * @description One multipart part named `icon`. The part name is fixed by the handler's
         *     `@RequestParam("icon")`; a different name is 400.
         */
        IconUpload: {
            /**
             * Format: binary
             * @description The icon bytes, at most 2 MB.
             */
            icon: string;
        };
        /**
         * @description The profile aggregate. Returned whole by every operation on this service
         *     except the icon download and the deletes, including the address and icon
         *     writes, which answer with the parent profile rather than the subresource.
         */
        Profile: {
            /** Format: int64 */
            id?: number;
            firstName?: string;
            middleName?: string | null;
            lastName?: string;
            /** Format: date-time */
            birthdate?: string;
            /** Format: int64 */
            userId?: number;
            addresses?: components["schemas"]["Address"][];
            /**
             * @description The icon's metadata and bytes, or null when the profile has none.
             *     Base64 in JSON, so a profile read carries the whole image; the
             *     dedicated `image/png` download exists for clients that want the bytes
             *     without the envelope.
             */
            icon?: components["schemas"]["ProfileIcon"] | null;
            /** Format: int64 */
            createdByUserId?: number;
            /** Format: date-time */
            createdTime?: string;
            /** Format: int64 */
            modifiedByUserId?: number;
            /** Format: date-time */
            modifiedTime?: string;
        };
        Address: {
            /** Format: int64 */
            id?: number;
            addressLine1?: string;
            addressLine2?: string | null;
            city?: string;
            stateProvince?: string;
            postalCode?: string;
            country?: string;
            /** Format: int64 */
            profileId?: number;
            /** Format: int64 */
            createdByUserId?: number;
            /** Format: date-time */
            createdTime?: string;
            /** Format: int64 */
            modifiedByUserId?: number;
            /** Format: date-time */
            modifiedTime?: string;
        };
        /**
         * @description Icon metadata plus the bytes. Identified by `profileId` — see
         *     `x-icon-identifier`.
         */
        ProfileIcon: {
            /**
             * Format: int64
             * @description Opaque surrogate key. No route accepts it and no lookup uses it;
             *     `profileId` is the icon's identifier. Present because a client type
             *     declares it.
             */
            id?: number;
            /**
             * Format: int64
             * @description The icon's identifier. At most one icon per profile.
             */
            profileId?: number;
            /** @description The stored bytes, base64-encoded in JSON. */
            icon?: string;
            /** Format: int64 */
            createdByUserId?: number;
            /** Format: date-time */
            createdTime?: string;
            /** Format: int64 */
            modifiedByUserId?: number;
            /** Format: date-time */
            modifiedTime?: string;
        };
        /** @description Field name to validation message, one entry per rejected field. */
        ValidationErrors: {
            [key: string]: string;
        };
        /**
         * @description The servlet container's error body, produced whenever a status is set
         *     through `sendError` rather than composed by a handler. Its exact keys are
         *     a framework detail; the frontends' shared error helper switches on the
         *     status code alone.
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
         *     through `HttpServletResponse.sendError`. `server.error.include-message`
         *     is unset, so Boot's default of `never` applies and no error page
         *     renders: the response carries no body and no `Content-Type`, for every
         *     `Accept` value. Clients key off the status and the header; there is no
         *     body to key off.
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
         * @description Authenticated but not permitted by the operation's `x-authorization`
         *     rule. Written through `sendError`, same as `Unauthorized` above — but
         *     unlike it, this one is not empty. The caller here already has a
         *     verified token, so the internal forward to `/error` that `sendError`
         *     triggers re-authenticates with that same token and reaches
         *     `BasicErrorController`, which renders Boot's standard error body.
         *     `message` is absent because `server.error.include-message` is
         *     `never`, not because the body itself is empty.
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
         *     `text/plain`, e.g. `Profile not found with id 42`.
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
         *     `text/plain`, e.g. `Profile already exists for user with id 7`.
         */
        Conflict: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "text/plain": string;
            };
        };
        /**
         * @description `IconUploadException` — the uploaded part could not be read. The body is
         *     the exception message as `text/plain`.
         */
        IconUploadFailed: {
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
        ProfileId: number;
        UserId: number;
        AddressId: number;
        /**
         * @description Declared explicitly because the handler reads the raw header to resolve
         *     the acting user for the audit columns. Redundant with the `bearerAuth`
         *     security scheme and kept only because removing it would change the served
         *     document. In the Go service the acting user comes from the verified
         *     `user_id` claim and this parameter disappears.
         */
        AuthorizationHeader: string;
    };
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    getProfiles: {
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
            /** @description A page of profiles as a bare array. No envelope, no total count. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"][];
                };
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    createProfile: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ProfileCreateRequestDTO"];
            };
        };
        responses: {
            /** @description Profile created. springdoc reports 200; the handler builds `HttpStatus.CREATED`. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
        };
    };
    getProfileById: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                profileId: components["parameters"]["ProfileId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The profile, with its addresses and icon metadata. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    updateProfileById: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                profileId: components["parameters"]["ProfileId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ProfileUpdateRequestDTO"];
            };
        };
        responses: {
            /** @description The updated profile. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    deleteProfileById: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                profileId: components["parameters"]["ProfileId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /**
             * @description Deleted, no body. springdoc reports `200` with a `Profile` body
             *     because the handler's declared return type is
             *     `ResponseEntity<Profile>`; the body it builds is `noContent()`.
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
    getProfileByUserId: {
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
            /** @description The profile. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    updateProfileByUserId: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
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
                "application/json": components["schemas"]["ProfileUpdateRequestDTO"];
            };
        };
        responses: {
            /** @description The updated profile. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    deleteProfileByUserId: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
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
            /** @description Deleted, no body. springdoc reports 200 with a `Profile` body; the handler builds `noContent()`. */
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
    addAddress: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                profileId: components["parameters"]["ProfileId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["AddressRequestDTO"];
            };
        };
        responses: {
            /** @description The profile with the new address in `addresses`. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    updateAddress: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                profileId: components["parameters"]["ProfileId"];
                addressId: components["parameters"]["AddressId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["AddressRequestDTO"];
            };
        };
        responses: {
            /** @description The profile with the updated address. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    deleteAddress: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                profileId: components["parameters"]["ProfileId"];
                addressId: components["parameters"]["AddressId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Deleted, no body. springdoc reports 200 with a `Profile` body; the handler builds `noContent()`. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            /**
             * @description The address does not belong to the profile in the path, so the
             *     scoped delete matched no row. `Address not found with id {addressId}
             *     for profile with id {profileId}` as `text/plain`. Unreachable for any
             *     caller acting on their own data.
             */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "text/plain": string;
                };
            };
        };
    };
    getProfileIcon: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                profileId: components["parameters"]["ProfileId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The icon bytes. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "image/png": string;
                };
            };
            400: components["responses"]["UnconvertableParameter"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            /** @description The profile has no icon. `Profile icon not found with id {profileId}` as `text/plain`. */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "text/plain": string;
                };
            };
            /**
             * @description The request's `Accept` excludes `image/png`, the only type this
             *     operation produces, so the mapping is dropped before the handler
             *     runs. The only 406 either service can answer: no other operation
             *     declares `produces`. The body is the container's — see
             *     `x-container-error`.
             */
            406: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ContainerError"];
                };
            };
        };
    };
    updateIcon: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                profileId: components["parameters"]["ProfileId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "multipart/form-data": components["schemas"]["IconUpload"];
            };
        };
        responses: {
            /** @description The profile with its replaced icon. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            /** @description The `icon` part is missing, or the upload is over the 2 MB cap. */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "text/plain": string;
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            /**
             * @description The upload could not be read (`IconUploadException`), or the profile
             *     had no icon to replace — see the description above. `text/plain` for
             *     the first, the container error body for the second.
             */
            500: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "text/plain": string;
                    "application/json": components["schemas"]["ContainerError"];
                };
            };
        };
    };
    addIcon: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                profileId: components["parameters"]["ProfileId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "multipart/form-data": components["schemas"]["IconUpload"];
            };
        };
        responses: {
            /** @description The profile with its new icon. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Profile"];
                };
            };
            /**
             * @description The `icon` part is missing
             *     (`MissingServletRequestPartException`), or the upload is over the 2 MB
             *     cap (`MultipartException`). Both answer `text/plain` with the
             *     exception's own message.
             */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "text/plain": string;
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            /** @description The profile already has an icon. Replace it with `PUT` instead. */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "text/plain": string;
                };
            };
            500: components["responses"]["IconUploadFailed"];
        };
    };
    deleteIcon: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description Declared explicitly because the handler reads the raw header to resolve
                 *     the acting user for the audit columns. Redundant with the `bearerAuth`
                 *     security scheme and kept only because removing it would change the served
                 *     document. In the Go service the acting user comes from the verified
                 *     `user_id` claim and this parameter disappears.
                 */
                Authorization: components["parameters"]["AuthorizationHeader"];
            };
            path: {
                profileId: components["parameters"]["ProfileId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Deleted, no body. springdoc reports 200 with a `Profile` body; the handler builds `noContent()`. */
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
}
