package com.jdw.usersrole.contracts;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.jdw.usersrole.models.SecurityUser;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.repositories.RoleRepository;
import com.jdw.usersrole.repositories.UserRepository;
import com.jdw.usersrole.services.JwtService;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.server.LocalServerPort;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.HttpStatus;
import org.springframework.http.HttpStatusCode;
import org.springframework.http.MediaType;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.web.client.RestClient;
import org.yaml.snakeyaml.Yaml;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.when;

/**
 * Asserts the half of the authorization contract that lives in the filter chain
 * rather than in an annotation.
 *
 * Six operations carry no `@PreAuthorize`, so `AuthorizationContractParityTests`
 * can only confirm that they carry none. What separates the two the contract
 * calls PUBLIC from the four it calls AUTHENTICATED is
 * `SecurityConfig`'s `.requestMatchers("/auth/**", "/actuator/**", "/openapi/**")
 * .permitAll()`, and nothing else in the build reads that matcher list. Flipping
 * it would leave every other check in this repository green while turning the
 * public endpoints private, or the private ones public.
 *
 * Rather than parse the matcher list — which asserts the source against itself —
 * this drives real unauthenticated requests and asserts the outcome the contract
 * claims: an operation marked PUBLIC must not answer 401, and every other
 * operation must.
 *
 * It also pins two shapes the frozen contracts document but the servlet layer
 * decides, not the annotations. `Unauthorized` carries no body and no
 * `Content-Type`: `CustomAuthenticationEntryPoint` calls `sendError`, and the
 * internal forward to `/error` that would normally render Boot's error body is
 * itself denied — still no token, so `AuthorizationFilter` rejects the
 * forwarded dispatch before `BasicErrorController` ever runs, and
 * `Access-Denied-Reason` is set a second time on the way. `Forbidden` reaches
 * no such dead end: the same verified token that got the caller past
 * authentication the first time authenticates the forwarded dispatch too, so
 * `BasicErrorController` runs normally and the body genuinely is Boot's
 * standard error JSON — this contract's `ContainerError`, transcribed rather
 * than assumed. A CORS preflight reaches `CorsFilter` and never the JWT
 * filter at all, so it answers 200 with no token.
 */
@Tag("fast")
@Tag("integration")
@SpringBootTest(
        webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT,
        // The default in application.yaml, `default-secret-key`, is not valid
        // base64 and only stands in until an environment supplies the real one.
        // A minted-token test needs a signing key that decodes; the value mirrors
        // JwtGoParityTests.PARITY_SECRET, a checked-in test fixture, not a
        // credential.
        properties = "app.jwt.secret-key=bXl0dGVzdHNlY3JldGtleWZvcmpzb253d2VidG9rZW4xMjM0NTY3ODkwIC1uCg==")
class FilterChainContractParityTests {
    private static final Path CONTRACTS = Path.of("docs", "contracts");
    private static final List<String> CONTRACT_FILES =
            List.of("identity-service.openapi.yaml", "profile-service.openapi.yaml");
    private static final List<String> HTTP_METHODS =
            List.of("get", "put", "post", "delete", "patch");
    private static final String PUBLIC_RULE = "PUBLIC";

    /** ADMIN-only, so any authenticated non-admin is refused, and answerable with no request body. */
    private static final String AN_ADMIN_ONLY_GET = "/api/users";

    private static final String A_PRINCIPAL_WITH_NO_AUTHORITIES = "forbidden-parity-test@example.com";

    @LocalServerPort
    private int port;

    @Autowired
    private JwtService jwtService;

    /**
     * Stand in for the database `JwtUserDetailService` reads on every request.
     * The 403 case needs a principal the JWT filter authenticates but method
     * security then rejects, and this suite boots no schema to authenticate one
     * against for real — see the class Javadoc.
     */
    @MockitoBean
    private UserRepository userRepository;

    @MockitoBean
    private RoleRepository roleRepository;

    private record Operation(HttpMethod method, String path, String rule) {
        String key() {
            return method.name() + " " + path;
        }
    }

    private record RawResponse(HttpStatusCode status, HttpHeaders headers, byte[] body) {
    }

    @Test
    void publicOperationsAreReachableWithoutATokenAndNothingElseIs() throws IOException {
        List<String> mismatches = new ArrayList<>();

        for (Operation operation : operations()) {
            HttpStatusCode status = callUnauthenticated(operation);
            boolean rejected = status.value() == HttpStatus.UNAUTHORIZED.value();

            if (PUBLIC_RULE.equals(operation.rule()) && rejected) {
                mismatches.add(operation.key()
                        + " is frozen as PUBLIC but the filter chain answers 401 without a token");
            }
            if (!PUBLIC_RULE.equals(operation.rule()) && !rejected) {
                mismatches.add(operation.key() + " is frozen as " + operation.rule()
                        + " but an unauthenticated request answers " + status.value() + ", not 401");
            }
        }

        assertTrue(mismatches.isEmpty(),
                "the filter chain and the frozen contracts disagree about what is reachable "
                        + "without a token:\n  " + String.join("\n  ", mismatches));
    }

    @Test
    void theContractsMarkExactlyTheAuthEndpointsPublic() throws IOException {
        List<String> publicOperations = operations().stream()
                .filter(operation -> PUBLIC_RULE.equals(operation.rule()))
                .map(Operation::key)
                .sorted()
                .toList();

        assertFalse(publicOperations.isEmpty(), "no operation is frozen as PUBLIC");
        assertTrue(publicOperations.stream().allMatch(key -> key.contains(" /auth/")),
                "an operation outside /auth is frozen as PUBLIC: " + publicOperations);
    }

    @Test
    void theAuthenticationEntryPointAnswersWithNoBodyAndNoContentType() {
        RawResponse response = call(HttpMethod.GET, AN_ADMIN_ONLY_GET, null);

        assertEquals(HttpStatus.UNAUTHORIZED.value(), response.status().value());
        assertEquals(0, response.body().length, "the 401 body is not empty");
        assertNull(response.headers().getContentType(), "the 401 response carries a Content-Type");
        assertEquals("Authentication Required", response.headers().getFirst("Access-Denied-Reason"));
    }

    /**
     * Unlike {@link #theAuthenticationEntryPointAnswersWithNoBodyAndNoContentType()},
     * this is not empty. The caller here is genuinely authenticated — only
     * {@code @PreAuthorize} refused it — so the same token authenticates the
     * internal forward to {@code /error} too, and {@code BasicErrorController}
     * runs to completion. `message` is absent rather than blank because
     * `server.error.include-message` is `never`, not merely unset to `""`.
     */
    @Test
    void theAccessDeniedHandlerAnswersWithTheContainerErrorBody() throws IOException {
        String token = jwtService.generateToken(principalWithNoAuthorities(), "https://parity-test.invalid");

        RawResponse response = call(HttpMethod.GET, AN_ADMIN_ONLY_GET, token);

        assertEquals(HttpStatus.FORBIDDEN.value(), response.status().value());
        assertEquals("Not Authorized", response.headers().getFirst("Access-Denied-Reason"));
        assertEquals(MediaType.APPLICATION_JSON, response.headers().getContentType());

        Map<?, ?> body = new ObjectMapper().readValue(response.body(), Map.class);
        assertEquals(403, body.get("status"));
        assertEquals("Forbidden", body.get("error"));
        assertTrue(body.containsKey("timestamp"), "the container error body has no timestamp");
        assertTrue(body.containsKey("path"), "the container error body has no path");
        assertFalse(body.containsKey("message"), "server.error.include-message is never, but message is present");
    }

    @Test
    void aPreflightToAnAuthenticatedPathAnswers200WithNoToken() {
        RawResponse response = preflight(AN_ADMIN_ONLY_GET);

        assertEquals(HttpStatus.OK.value(), response.status().value(),
                "a preflight to an authenticated path did not bypass the JWT filter");
    }

    /**
     * A body is sent wherever the operation declares one, so a permitted request
     * fails validation rather than the body being missing — either way it is not
     * a 401, which is the only thing under test. Path variables are filled with a
     * value that parses as a long so routing is never the reason for a 4xx.
     */
    private HttpStatusCode callUnauthenticated(Operation operation) {
        String uri = "http://localhost:" + port
                + operation.path().replaceAll("\\{[^}]+}", "1");
        RestClient.RequestBodySpec request = RestClient.create()
                .method(operation.method())
                .uri(uri);
        request.contentType(MediaType.APPLICATION_JSON).body("{}");
        return request.exchange((sent, response) -> response.getStatusCode(), false);
    }

    /** No request body: both callers hit an operation whose `@PreAuthorize` runs before any body is read. */
    private RawResponse call(HttpMethod method, String path, String bearerToken) {
        RestClient.RequestHeadersSpec<?> request = RestClient.create()
                .method(method)
                .uri("http://localhost:" + port + path)
                .headers(headers -> {
                    if (bearerToken != null) {
                        headers.setBearerAuth(bearerToken);
                    }
                });
        return request.exchange((sent, response) -> new RawResponse(
                response.getStatusCode(), response.getHeaders(), response.getBody().readAllBytes()), true);
    }

    /**
     * The three conditions Spring's `CorsUtils.isPreFlightRequest` requires:
     * `OPTIONS`, an `Origin`, and `Access-Control-Request-Method`. No
     * `Authorization` header is sent — a browser never attaches one to a
     * preflight, and this is the case the contract says never reaches the JWT
     * filter at all.
     */
    private RawResponse preflight(String path) {
        RestClient.RequestHeadersSpec<?> request = RestClient.create()
                .method(HttpMethod.OPTIONS)
                .uri("http://localhost:" + port + path)
                .header(HttpHeaders.ORIGIN, "https://parity-test.invalid")
                .header("Access-Control-Request-Method", "GET");
        return request.exchange((sent, response) -> new RawResponse(
                response.getStatusCode(), response.getHeaders(), response.getBody().readAllBytes()), true);
    }

    /**
     * `SecurityUser.getAuthorities()` maps `user.roles()` through
     * `roleRepository`; an empty role set short-circuits that entirely, so the
     * mocked `RoleRepository` is never called and needs no stubbing.
     */
    private SecurityUser principalWithNoAuthorities() {
        User user = User.builder()
                .id(-1L)
                .emailAddress(A_PRINCIPAL_WITH_NO_AUTHORITIES)
                .password("unused")
                .status("ACTIVE")
                .roles(Set.of())
                .build();
        when(userRepository.findByEmailAddress(A_PRINCIPAL_WITH_NO_AUTHORITIES)).thenReturn(Optional.of(user));
        return new SecurityUser(user, roleRepository);
    }

    private List<Operation> operations() throws IOException {
        List<Operation> operations = new ArrayList<>();
        for (String file : CONTRACT_FILES) {
            Map<String, Object> document = new Yaml().load(Files.readString(CONTRACTS.resolve(file)));
            Map<?, ?> paths = (Map<?, ?>) document.get("paths");
            paths.forEach((path, item) -> ((Map<?, ?>) item).forEach((field, value) -> {
                if (!HTTP_METHODS.contains(field.toString().toLowerCase())) {
                    return;
                }
                Map<?, ?> operation = (Map<?, ?>) value;
                Map<?, ?> authorization = (Map<?, ?>) operation.get("x-authorization");
                operations.add(new Operation(
                        HttpMethod.valueOf(field.toString().toUpperCase()),
                        path.toString(),
                        String.valueOf(authorization.get("rule"))));
            }));
        }
        return operations;
    }
}
