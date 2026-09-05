package com.jdw.usersrole.contracts;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.server.LocalServerPort;
import org.springframework.http.HttpMethod;
import org.springframework.http.HttpStatus;
import org.springframework.http.HttpStatusCode;
import org.springframework.http.MediaType;
import org.springframework.web.client.RestClient;
import org.yaml.snakeyaml.Yaml;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

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
 */
@Tag("fast")
@Tag("integration")
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
class FilterChainContractParityTests {
    private static final Path CONTRACTS = Path.of("docs", "contracts");
    private static final List<String> CONTRACT_FILES =
            List.of("identity-service.openapi.yaml", "profile-service.openapi.yaml");
    private static final List<String> HTTP_METHODS =
            List.of("get", "put", "post", "delete", "patch");
    private static final String PUBLIC_RULE = "PUBLIC";

    @LocalServerPort
    private int port;

    private record Operation(HttpMethod method, String path, String rule) {
        String key() {
            return method.name() + " " + path;
        }
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
