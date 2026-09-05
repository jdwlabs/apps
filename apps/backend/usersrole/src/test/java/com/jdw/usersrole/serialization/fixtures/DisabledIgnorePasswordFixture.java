package com.jdw.usersrole.serialization.fixtures;

import com.fasterxml.jackson.annotation.JsonIgnore;

public record DisabledIgnorePasswordFixture(
        Long id,
        @JsonIgnore(false)
        String password
) {
}
