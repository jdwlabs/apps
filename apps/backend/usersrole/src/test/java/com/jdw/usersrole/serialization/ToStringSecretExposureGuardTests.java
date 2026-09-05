package com.jdw.usersrole.serialization;

import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.serialization.fixtures.NestedUnmaskedPasswordFixture;
import com.jdw.usersrole.serialization.fixtures.UnmaskedPasswordFixture;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.context.annotation.ClassPathScanningCandidateComponentProvider;

import java.lang.reflect.Constructor;
import java.lang.reflect.RecordComponent;
import java.util.ArrayList;
import java.util.HashSet;
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

    // A secret does not have to be the outer type's own field to leak: User.profile() embeds
    // Profile directly, so a secret added a level down would render into the outer toString()
    // through plain string concatenation just as easily as one at the top level.
    @Test
    void guardFlagsAToStringThatLeaksASecretThroughANestedRecord() {
        assertEquals(List.of(NestedUnmaskedPasswordFixture.class.getName()),
                leakedToStrings(Set.of(NestedUnmaskedPasswordFixture.class)));
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
        return instantiateWithSecretMarker(type, new HashSet<>());
    }

    // `visiting` tracks the current instantiation chain (not every type seen), so a record
    // that legitimately embeds the same nested type twice in different branches is still
    // built in full — only an actual cycle (a type embedding itself, directly or through
    // others) gets short-circuited to null.
    private Object instantiateWithSecretMarker(Class<?> type, Set<Class<?>> visiting) {
        if (!visiting.add(type)) {
            return null;
        }
        try {
            RecordComponent[] components = type.getRecordComponents();
            Class<?>[] paramTypes = new Class<?>[components.length];
            Object[] args = new Object[components.length];
            for (int i = 0; i < components.length; i++) {
                paramTypes[i] = components[i].getType();
                args[i] = valueFor(components[i], visiting);
            }
            Constructor<?> constructor = type.getDeclaredConstructor(paramTypes);
            constructor.setAccessible(true);
            return constructor.newInstance(args);
        } catch (ReflectiveOperationException e) {
            throw new IllegalStateException(
                    "could not instantiate " + type.getName() + " via its canonical constructor", e);
        } finally {
            visiting.remove(type);
        }
    }

    private Object valueFor(RecordComponent component, Set<Class<?>> visiting) {
        Class<?> type = component.getType();
        if (namesASecret(component.getName()) && type == String.class) {
            return SECRET_MARKER;
        }
        if (isRecursableRecord(type)) {
            return instantiateWithSecretMarker(type, visiting);
        }
        return defaultValueFor(type);
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
                    .filter(type -> hasSecretField(type, new HashSet<>()))
                    .forEach(types::add);
        }
        return types;
    }

    // Recurses into nested record components (e.g. User -> Profile) so a secret does not have
    // to be a direct field of the scanned type to be caught — it only has to be reachable
    // through toString()'s implicit chain of nested calls. `visited` here is a plain
    // once-ever set rather than an ancestor stack: whether a type transitively reaches a
    // secret is a fixed property of that type, independent of which path found it, so
    // short-circuiting a repeat visit (including a cyclic one) never hides a real positive.
    private boolean hasSecretField(Class<?> type, Set<Class<?>> visited) {
        if (!visited.add(type)) {
            return false;
        }
        for (RecordComponent component : type.getRecordComponents()) {
            if (namesASecret(component.getName())) {
                return true;
            }
            Class<?> componentType = component.getType();
            if (isRecursableRecord(componentType) && hasSecretField(componentType, visited)) {
                return true;
            }
        }
        return false;
    }

    private boolean isRecursableRecord(Class<?> type) {
        return type.isRecord() && !isJdkType(type);
    }

    private boolean isJdkType(Class<?> type) {
        String packageName = type.getPackageName();
        return packageName.startsWith("java.") || packageName.startsWith("javax.");
    }

    private Class<?> loadClass(String className) {
        try {
            return Class.forName(className);
        } catch (ClassNotFoundException e) {
            throw new IllegalStateException("scanned class is not loadable: " + className, e);
        }
    }
}
