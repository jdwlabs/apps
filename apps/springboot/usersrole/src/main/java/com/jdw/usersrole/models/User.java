package com.jdw.usersrole.models;

import lombok.Builder;

import java.sql.Timestamp;
import java.util.Set;

@Builder
public record User(
        Long id,
        String emailAddress,
        String password,
        String status,
        Set<UserRole> roles,
        Profile profile,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
