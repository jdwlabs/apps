package com.jdw.usersrole.models;

import org.springframework.data.jdbc.core.mapping.AggregateReference;
import org.springframework.data.relational.core.mapping.Table;

import java.sql.Timestamp;

@Table(value = "auth\".\"users_roles")
public record UserRole(
//        AggregateReference<User, Long> userId,
        Long userId,
//        AggregateReference<Role, Long> roleId,
        Long roleId,
        Long createdByUserId,
        Timestamp createdTime
) {
}
