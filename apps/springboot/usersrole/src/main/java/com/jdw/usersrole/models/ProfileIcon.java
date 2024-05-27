package com.jdw.usersrole.models;

import lombok.Builder;
import org.springframework.data.annotation.Id;
import org.springframework.data.jdbc.core.mapping.AggregateReference;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.Table;

import java.sql.Timestamp;

@Builder
@Table(value = "auth\".\"profile_icons")
public record ProfileIcon(
        @Id @Column("icon_id") Long id,
        @Column("profile_id") AggregateReference<Profile, Long> profileId,
//        @Column("profile_id") Long profileId,
        byte[] icon,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
