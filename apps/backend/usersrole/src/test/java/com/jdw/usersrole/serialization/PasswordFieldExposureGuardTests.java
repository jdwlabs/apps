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

import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag("fast")
@Tag("unit")
class PasswordFieldExposureGuardTests {
    private static final List<String> SERIALIZABLE_PACKAGES =
            List.of("com.jdw.usersrole.models", "com.jdw.usersrole.dtos");

    @Test
    void everyPasswordFieldIsWithheldFromSerialization() {
        Set<Class<?>> candidates = serializableTypes();

        assertTrue(candidates.containsAll(Set.of(User.class, UserRequestDTO.class)),
                "the scan missed known password-carrying types, so this guard would pass vacuously: " + candidates);

        List<String> exposed = new ArrayList<>();
        for (Class<?> type : candidates) {
            for (Field field : type.getDeclaredFields()) {
                if (!"password".equalsIgnoreCase(field.getName()) || withheld(type, field)) {
                    continue;
                }
                exposed.add(type.getName() + "#" + field.getName());
            }
        }

        assertTrue(exposed.isEmpty(),
                "password fields must carry @JsonIgnore or @JsonProperty(access = WRITE_ONLY): " + exposed);
    }

    private boolean withheld(Class<?> type, Field field) {
        return accessorsFor(type, field)
                .anyMatch(element -> element.isAnnotationPresent(JsonIgnore.class) || isWriteOnly(element));
    }

    private Stream<AnnotatedElement> accessorsFor(Class<?> type, Field field) {
        Stream<AnnotatedElement> element = Stream.of(field);
        try {
            Method accessor = type.getDeclaredMethod(field.getName());
            return Stream.concat(element, Stream.of(accessor));
        } catch (NoSuchMethodException e) {
            return element;
        }
    }

    private boolean isWriteOnly(AnnotatedElement element) {
        JsonProperty jsonProperty = element.getAnnotation(JsonProperty.class);
        return jsonProperty != null && jsonProperty.access() == JsonProperty.Access.WRITE_ONLY;
    }

    private Set<Class<?>> serializableTypes() {
        ClassPathScanningCandidateComponentProvider scanner =
                new ClassPathScanningCandidateComponentProvider(false);
        scanner.addIncludeFilter((metadataReader, metadataReaderFactory) -> true);

        Set<Class<?>> types = new LinkedHashSet<>();
        for (String basePackage : SERIALIZABLE_PACKAGES) {
            scanner.findCandidateComponents(basePackage).stream()
                    .map(definition -> definition.getBeanClassName())
                    .map(this::loadClass)
                    // Nested types here are only Lombok's generated builders, which are
                    // construction scaffolding and never reach a response body.
                    .filter(type -> type.getEnclosingClass() == null)
                    .forEach(types::add);
        }
        return types;
    }

    private Class<?> loadClass(String className) {
        try {
            return Class.forName(className);
        } catch (ClassNotFoundException e) {
            throw new IllegalStateException("scanned class is not loadable: " + className, e);
        }
    }
}
