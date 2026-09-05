package com.jdw.usersrole.serialization.fixtures;

import com.fasterxml.jackson.annotation.JsonProperty;

public record WriteOnlyPasswordFixture(
        Long id,
        @JsonProperty(access = JsonProperty.Access.WRITE_ONLY)
        String password
) {
}
