package com.jdw.usersrole.contracts;

import com.jdw.usersrole.controllers.AuthController;
import com.jdw.usersrole.controllers.ProfilesController;
import com.jdw.usersrole.controllers.RolesController;
import com.jdw.usersrole.controllers.UsersController;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.yaml.snakeyaml.Yaml;

import java.io.IOException;
import java.lang.annotation.Annotation;
import java.lang.reflect.Method;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

/**
 * Holds the frozen contracts to the authorization the controllers actually
 * apply. The contracts are what the Go services are built and parity-tested
 * against, so an `x-authorization` block that has drifted from its
 * `@PreAuthorize` predicate would hand the rewrite a wrong answer with nothing
 * downstream able to notice.
 *
 * The drift checker compares the contracts to the served springdoc document,
 * which carries no authorization information at all — this is the half of the
 * contract that check cannot see.
 */
@Tag("fast")
class AuthorizationContractParityTests {
    private static final Path CONTRACTS = Path.of("docs", "contracts");
    private static final List<Class<?>> CONTROLLERS =
            List.of(AuthController.class, UsersController.class, RolesController.class, ProfilesController.class);
    private static final int EXPECTED_MAPPINGS = 33;
    private static final List<String> HTTP_METHODS =
            List.of("get", "put", "post", "delete", "patch", "head", "options", "trace");

    private record Mapping(String httpMethod, String path, String preAuthorize) {
        String key() {
            return httpMethod + " " + path;
        }
    }

    @Test
    void everyMappingIsFrozenWithTheAuthorizationItApplies() throws IOException {
        Map<String, Map<String, Object>> frozen = frozenOperations();
        List<Mapping> mappings = mappings();

        assertEquals(EXPECTED_MAPPINGS, mappings.size(),
                "the controllers no longer declare " + EXPECTED_MAPPINGS + " mappings: " + mappings);

        List<String> mismatches = new ArrayList<>();
        for (Mapping mapping : mappings) {
            Map<String, Object> operation = frozen.get(mapping.key());
            if (operation == null) {
                mismatches.add(mapping.key() + " is mapped by a controller but frozen in neither contract");
                continue;
            }
            Object authorization = operation.get("x-authorization");
            if (!(authorization instanceof Map<?, ?> block)) {
                mismatches.add(mapping.key() + " carries no x-authorization block");
                continue;
            }
            if (!block.containsKey("preAuthorize")) {
                // Without this, `preAuthorize: null` and a deleted key are the same
                // value, and the six operations that carry no predicate would be
                // asserted vacuously.
                mismatches.add(mapping.key() + " has an x-authorization block with no preAuthorize key");
                continue;
            }
            Object frozenPredicate = block.get("preAuthorize");
            if (!java.util.Objects.equals(mapping.preAuthorize(), frozenPredicate)) {
                mismatches.add(mapping.key()
                        + " applies " + describe(mapping.preAuthorize())
                        + " but the contract freezes " + describe(frozenPredicate));
            }
        }

        for (String key : frozen.keySet()) {
            if (mappings.stream().noneMatch(mapping -> mapping.key().equals(key))) {
                mismatches.add(key + " is frozen in a contract but no controller maps it");
            }
        }

        assertTrue(mismatches.isEmpty(),
                "the frozen contracts and the controllers disagree about authorization:\n  "
                        + String.join("\n  ", mismatches));
    }

    private static String describe(Object predicate) {
        return predicate == null ? "no @PreAuthorize (filter chain only)" : "@PreAuthorize(\"" + predicate + "\")";
    }

    private Map<String, Map<String, Object>> frozenOperations() throws IOException {
        Map<String, Map<String, Object>> operations = new LinkedHashMap<>();
        for (String file : List.of("identity-service.openapi.yaml", "profile-service.openapi.yaml")) {
            Map<String, Object> document = new Yaml().load(Files.readString(CONTRACTS.resolve(file)));
            Map<?, ?> paths = (Map<?, ?>) document.get("paths");
            paths.forEach((path, item) -> ((Map<?, ?>) item).forEach((field, value) -> {
                // A path item may also carry `parameters`, `summary` and the like,
                // which are not operations and are not Maps.
                if (!HTTP_METHODS.contains(field.toString().toLowerCase())) {
                    return;
                }
                @SuppressWarnings("unchecked")
                Map<String, Object> operation = (Map<String, Object>) value;
                String key = field.toString().toUpperCase() + " " + path;
                assertTrue(operations.put(key, operation) == null, key + " is frozen in both contracts");
            }));
        }
        return operations;
    }

    private List<Mapping> mappings() {
        List<Mapping> mappings = new ArrayList<>();
        for (Class<?> controller : CONTROLLERS) {
            String base = firstPath(controller.getAnnotation(RequestMapping.class).value());
            for (Method method : controller.getDeclaredMethods()) {
                httpMethodOf(method).ifPresent(httpMethod -> {
                    PreAuthorize preAuthorize = method.getAnnotation(PreAuthorize.class);
                    mappings.add(new Mapping(
                            httpMethod,
                            base + firstPath(pathOf(method)),
                            preAuthorize == null ? null : preAuthorize.value()));
                });
            }
        }
        mappings.sort(Comparator.comparing(Mapping::key));
        return mappings;
    }

    private static java.util.Optional<String> httpMethodOf(Method method) {
        for (Map.Entry<Class<? extends Annotation>, String> candidate : Map.<Class<? extends Annotation>, String>of(
                GetMapping.class, "GET",
                PostMapping.class, "POST",
                PutMapping.class, "PUT",
                DeleteMapping.class, "DELETE",
                PatchMapping.class, "PATCH").entrySet()) {
            if (method.isAnnotationPresent(candidate.getKey())) {
                return java.util.Optional.of(candidate.getValue());
            }
        }
        return java.util.Optional.empty();
    }

    private static String[] pathOf(Method method) {
        if (method.isAnnotationPresent(GetMapping.class)) return method.getAnnotation(GetMapping.class).value();
        if (method.isAnnotationPresent(PostMapping.class)) return method.getAnnotation(PostMapping.class).value();
        if (method.isAnnotationPresent(PutMapping.class)) return method.getAnnotation(PutMapping.class).value();
        if (method.isAnnotationPresent(DeleteMapping.class)) return method.getAnnotation(DeleteMapping.class).value();
        return method.getAnnotation(PatchMapping.class).value();
    }

    private static String firstPath(String[] paths) {
        return paths.length == 0 ? "" : paths[0];
    }
}
