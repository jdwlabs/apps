package com.jdw.usersrole.models;

import lombok.Builder;

import java.sql.Timestamp;

@Builder
public record Address(
        Long id,
        String addressLine1,
        String addressLine2,
        String city,
        String stateProvince,
        String postalCode,
        String country,
        Long profileId,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
