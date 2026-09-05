package com.jdw.usersrole.serialization.fixtures;

import com.fasterxml.jackson.annotation.JsonIgnore;

// @JsonIgnore keeps this out of the sibling JSON-exposure guard's findings (it is withheld
// from serialization, same as the real UserRequestDTO/User), but a record's canonical
// toString() emits every component regardless of Jackson annotations. That gap is exactly
// the bug this ticket fixes, so this fixture must be flagged by the toString exposure guard,
// proving the guard still detects a leak rather than passing vacuously.
public record UnmaskedPasswordFixture(
        Long id,
        @JsonIgnore
        String password
) {
}
