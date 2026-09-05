package com.jdw.usersrole.serialization.fixtures;

import com.fasterxml.jackson.annotation.JsonIgnore;

public record WithheldPasswordFixture(
        Long id,
        @JsonIgnore
        String password
) {
}
