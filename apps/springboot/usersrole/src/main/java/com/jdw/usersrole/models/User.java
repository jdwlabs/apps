package com.jdw.usersrole.models;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.MappedCollection;
import org.springframework.data.relational.core.mapping.Table;

import java.sql.Timestamp;
import java.util.Set;

@Table(value = "auth\".\"users")
public record User(
        @Id @Column("user_id") Long id,
        String emailAddress,
        String password,
        String status,
        @MappedCollection(idColumn = "user_id") Set<UserRole> roles,
        @MappedCollection(idColumn = "user_id", keyColumn = "user_id") Profile profile,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
}
