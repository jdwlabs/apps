package com.jdw.usersrole.models;

import lombok.Builder;

import java.sql.Timestamp;

@Builder
public record ProfileIcon(
        Long id,
        Long profileId,
        byte[] icon,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
