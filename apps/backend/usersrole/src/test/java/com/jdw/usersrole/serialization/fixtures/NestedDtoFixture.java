package com.jdw.usersrole.serialization.fixtures;

public record NestedDtoFixture(Long id, Nested nested) {
    public record Nested(String password) {
    }
}
