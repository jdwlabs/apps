package com.jdw.usersrole.models;

import lombok.Builder;

import java.sql.Timestamp;
import java.util.Set;

@Builder
public record Role(
        Long id,
        String name,
        String description,
        String status,
        Set<UserRole> users,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
