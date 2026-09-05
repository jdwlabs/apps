package com.jdw.usersrole.serialization.fixtures;

import com.fasterxml.jackson.annotation.JsonIgnore;

// The outer record carries no secret field of its own; the secret is only reachable through
// the nested record it embeds — mirrors User.profile() in production. Proves the toString
// exposure guard recurses into nested record components instead of checking only the outer
// type's own fields. @JsonIgnore on the inner password keeps this from also tripping the
// sibling JSON-exposure guard, which covers a different, already-verified concern.
public record NestedUnmaskedPasswordFixture(Long id, Nested nested) {
    public record Nested(
            Long id,
            @JsonIgnore
            String password
    ) {
    }
}
