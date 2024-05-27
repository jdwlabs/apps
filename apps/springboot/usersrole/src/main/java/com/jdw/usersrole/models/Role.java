package com.jdw.usersrole.models;

import lombok.Builder;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.MappedCollection;
import org.springframework.data.relational.core.mapping.Table;

import java.sql.Timestamp;
import java.util.Set;

@Builder
@Table(value = "auth\".\"roles")
public record Role(
        @Id @Column("role_id") Long id,
        @Column("role_name") String name,
        @Column("role_description") String description,
        String status,
        @MappedCollection(idColumn = "role_id") Set<UserRole> users,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
