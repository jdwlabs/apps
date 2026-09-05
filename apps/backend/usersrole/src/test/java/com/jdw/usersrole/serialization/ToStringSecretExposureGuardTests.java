package com.jdw.usersrole.serialization;

import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.serialization.fixtures.UnmaskedPasswordFixture;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.context.annotation.ClassPathScanningCandidateComponentProvider;

import java.lang.reflect.Constructor;
import java.lang.reflect.RecordComponent;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

// Sibling to PasswordFieldExposureGuardTests: that guard proves secret fields are withheld
// from Jackson serialization, this one proves they are withheld from toString() — a
// completely separate code path that Jackson annotations do not touch, and the one a
// TRACE-level log statement actually goes through.
@Tag("fast")
@Tag("unit")
class ToStringSecretExposureGuardTests {
    private static final List<String> SERIALIZABLE_PACKAGES =
            List.of("com.jdw.usersrole.models", "com.jdw.usersrole.dtos");

    private static final List<String> SECRET_NAME_MARKERS = List.of("password", "secret");

    // Unique per run so a "leak" can only be this test's own injected value making it
    // through toString(), never a coincidental match against unrelated literal text.
    private static final String SECRET_MARKER = "S3cr3tMarker-" + System.nanoTime();

    @Test
    void everySecretFieldIsMaskedFromToString() {
        Set<Class<?>> scanned = recordTypesWithSecretFields(SERIALIZABLE_PACKAGES);

        assertTrue(scanned.containsAll(Set.of(User.class, UserRequestDTO.class)),
                "the scan missed known secret-carrying types, so this guard would pass vacuously: " + scanned);

        assertEquals(List.of(), leakedToStrings(scanned),
                "toString() must not emit a field named like a secret");
    }

    // Proves the guard still bites, so a green run above means the toString() overrides
    // hold rather than the detection having quietly stopped working.
    @Test
    void guardFlagsAToStringThatStillLeaksTheSecret() {
        assertEquals(List.of(UnmaskedPasswordFixture.class.getName()),
                leakedToStrings(Set.of(UnmaskedPasswordFixture.class)));
    }

    private List<String> leakedToStrings(Set<Class<?>> types) {
        List<String> leaked = new ArrayList<>();
        for (Class<?> type : types) {
            Object instance = instantiateWithSecretMarker(type);
            if (instance.toString().contains(SECRET_MARKER)) {
                leaked.add(type.getName());
            }
        }
        return leaked.stream().sorted().toList();
    }

    private Object instantiateWithSecretMarker(Class<?> type) {
        RecordComponent[] components = type.getRecordComponents();
        Class<?>[] paramTypes = new Class<?>[components.length];
        Object[] args = new Object[components.length];
        for (int i = 0; i < components.length; i++) {
            paramTypes[i] = components[i].getType();
            args[i] = valueFor(components[i]);
        }
        try {
            Constructor<?> constructor = type.getDeclaredConstructor(paramTypes);
            constructor.setAccessible(true);
            return constructor.newInstance(args);
        } catch (ReflectiveOperationException e) {
            throw new IllegalStateException(
                    "could not instantiate " + type.getName() + " via its canonical constructor", e);
        }
    }

    private Object valueFor(RecordComponent component) {
        if (namesASecret(component.getName()) && component.getType() == String.class) {
            return SECRET_MARKER;
        }
        return defaultValueFor(component.getType());
    }

    private Object defaultValueFor(Class<?> type) {
        if (!type.isPrimitive()) {
            return null;
        }
        if (type == boolean.class) {
            return false;
        }
        if (type == char.class) {
            return '\0';
        }
        if (type == byte.class) {
            return (byte) 0;
        }
        if (type == short.class) {
            return (short) 0;
        }
        if (type == int.class) {
            return 0;
        }
        if (type == long.class) {
            return 0L;
        }
        if (type == float.class) {
            return 0f;
        }
        return 0d;
    }

    private boolean namesASecret(String name) {
        String lower = name.toLowerCase();
        return SECRET_NAME_MARKERS.stream().anyMatch(lower::contains);
    }

    private Set<Class<?>> recordTypesWithSecretFields(List<String> basePackages) {
        ClassPathScanningCandidateComponentProvider scanner =
                new ClassPathScanningCandidateComponentProvider(false);
        scanner.addIncludeFilter((metadataReader, metadataReaderFactory) -> true);

        Set<Class<?>> types = new LinkedHashSet<>();
        for (String basePackage : basePackages) {
            scanner.findCandidateComponents(basePackage).stream()
                    .map(definition -> definition.getBeanClassName())
                    .map(this::loadClass)
                    .filter(Class::isRecord)
                    .filter(this::hasSecretField)
                    .forEach(types::add);
        }
        return types;
    }

    private boolean hasSecretField(Class<?> type) {
        for (RecordComponent component : type.getRecordComponents()) {
            if (namesASecret(component.getName())) {
                return true;
            }
        }
        return false;
    }

    private Class<?> loadClass(String className) {
        try {
            return Class.forName(className);
        } catch (ClassNotFoundException e) {
            throw new IllegalStateException("scanned class is not loadable: " + className, e);
        }
    }
}
