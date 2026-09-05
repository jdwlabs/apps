package com.jdw.usersrole.dtos;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertFalse;

@Tag("fast")
@Tag("unit")
class UserRequestDTOTests {
    @Test
    void toString_neverContainsTheCleartextPassword() {
        String knownPassword = "P@ssw0rd123!";
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", knownPassword);

        String rendered = userRequestDTO.toString();

        assertFalse(rendered.contains(knownPassword),
                "UserRequestDTO.toString() must mask the password: " + rendered);
    }
}
