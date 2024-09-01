package com.jdw.usersrole.models;

import lombok.Builder;

import java.sql.Date;
import java.sql.Timestamp;
import java.util.Set;

@Builder
public record Profile(
        Long id,
        String firstName,
        String middleName,
        String lastName,
        Date birthdate,
        Long userId,
        Set<Address> addresses,
        ProfileIcon icon,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
