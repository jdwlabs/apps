package com.jdw.usersrole.models;

import lombok.Builder;

import java.sql.Timestamp;

@Builder
public record UserRole(
        Long userId,
        Long roleId,
        Long createdByUserId,
        Timestamp createdTime
) {
}
