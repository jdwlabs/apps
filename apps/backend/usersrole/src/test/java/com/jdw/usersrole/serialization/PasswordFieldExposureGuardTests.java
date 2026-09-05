package com.jdw.usersrole.serialization;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.context.annotation.ClassPathScanningCandidateComponentProvider;

import java.lang.reflect.AnnotatedElement;
import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
import java.util.stream.Stream;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag("fast")
@Tag("unit")
class PasswordFieldExposureGuardTests {
    private static final List<String> SERIALIZABLE_PACKAGES =
            List.of("com.jdw.usersrole.models", "com.jdw.usersrole.dtos");
    private static final String FIXTURE_PACKAGE = "com.jdw.usersrole.serialization.fixtures";

    // Matched as substrings, because passwordHash, newPassword and currentPassword are
    // the same secret under a different name. "token" is deliberately absent: the whole
    // purpose of AuthResponseDTO is to serialize jwtToken, so that marker would need an
    // allowlist beside it and would earn nothing the two below do not.
    private static final List<String> SECRET_NAME_MARKERS = List.of("password", "secret");

    @Test
    void everyPasswordFieldIsWithheldFromSerialization() {
        Set<Class<?>> scanned = serializableTypes(SERIALIZABLE_PACKAGES);

        assertTrue(scanned.containsAll(Set.of(User.class, UserRequestDTO.class)),
                "the scan missed known secret-carrying types, so this guard would pass vacuously: " + scanned);

        assertEquals(List.of(), exposedFieldsIn(scanned),
                "secret-bearing fields must carry @JsonIgnore or @JsonProperty(access = WRITE_ONLY)");
    }

    // Proves the guard still bites, so a green run above means the annotations hold rather
    // than the detection having quietly stopped working.
    @Test
    void guardFlagsSecretsTheAnnotationsDoNotWithhold() {
        Set<Class<?>> fixtures = serializableTypes(List.of(FIXTURE_PACKAGE));

        assertEquals(List.of(
                        FIXTURE_PACKAGE + ".ApiSecretFixture#apiSecret",
                        FIXTURE_PACKAGE + ".DerivedNamePasswordFixture#passwordHash",
                        FIXTURE_PACKAGE + ".DisabledIgnorePasswordFixture#password",
                        FIXTURE_PACKAGE + ".NestedDtoFixture$Nested#password"),
                exposedFieldsIn(fixtures));
    }

    private List<String> exposedFieldsIn(Set<Class<?>> types) {
        List<String> exposed = new ArrayList<>();
        for (Class<?> type : types) {
            for (Field field : type.getDeclaredFields()) {
                if (!namesASecret(field) || withheld(type, field)) {
                    continue;
                }
                exposed.add(type.getName() + "#" + field.getName());
            }
        }
        return exposed.stream().sorted().toList();
    }

    private boolean namesASecret(Field field) {
        String name = field.getName().toLowerCase();
        return SECRET_NAME_MARKERS.stream().anyMatch(name::contains);
    }

    private boolean withheld(Class<?> type, Field field) {
        return declarationsOf(type, field).anyMatch(this::isIgnored);
    }

    private Stream<AnnotatedElement> declarationsOf(Class<?> type, Field field) {
        Stream<AnnotatedElement> declarations = Stream.of(field);
        try {
            Method accessor = type.getDeclaredMethod(field.getName());
            return Stream.concat(declarations, Stream.of(accessor));
        } catch (NoSuchMethodException e) {
            return declarations;
        }
    }

    private boolean isIgnored(AnnotatedElement element) {
        // @JsonIgnore(false) means "not ignored", so presence alone is not withholding.
        JsonIgnore jsonIgnore = element.getAnnotation(JsonIgnore.class);
        if (jsonIgnore != null && jsonIgnore.value()) {
            return true;
        }
        JsonProperty jsonProperty = element.getAnnotation(JsonProperty.class);
        return jsonProperty != null && jsonProperty.access() == JsonProperty.Access.WRITE_ONLY;
    }

    private Set<Class<?>> serializableTypes(List<String> basePackages) {
        ClassPathScanningCandidateComponentProvider scanner =
                new ClassPathScanningCandidateComponentProvider(false);
        scanner.addIncludeFilter((metadataReader, metadataReaderFactory) -> true);

        Set<Class<?>> types = new LinkedHashSet<>();
        for (String basePackage : basePackages) {
            scanner.findCandidateComponents(basePackage).stream()
                    .map(definition -> definition.getBeanClassName())
                    .map(this::loadClass)
                    .filter(type -> !isGeneratedBuilder(type))
                    .forEach(types::add);
        }
        return types;
    }

    // Lombok's @Builder emits a nested XBuilder holding a copy of every component. It is
    // construction scaffolding that never reaches a response body, and it cannot carry the
    // annotations anyway. Every other nested type stays in scope.
    private boolean isGeneratedBuilder(Class<?> type) {
        return type.getEnclosingClass() != null && type.getSimpleName().endsWith("Builder");
    }

    private Class<?> loadClass(String className) {
        try {
            return Class.forName(className);
        } catch (ClassNotFoundException e) {
            throw new IllegalStateException("scanned class is not loadable: " + className, e);
        }
    }
}
