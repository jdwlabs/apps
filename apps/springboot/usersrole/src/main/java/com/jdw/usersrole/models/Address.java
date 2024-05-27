package com.jdw.usersrole.models;

import lombok.Builder;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.Table;

import java.sql.Timestamp;

@Builder
@Table(value = "auth\".\"addresses")
public record Address(
        @Id @Column("address_id") Long id,
        String addressLine1,
        String addressLine2,
        String city,
        String stateProvince,
        String postalCode,
        String country,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
