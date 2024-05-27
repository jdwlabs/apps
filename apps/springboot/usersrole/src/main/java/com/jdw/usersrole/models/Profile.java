package com.jdw.usersrole.models;

import lombok.Builder;
import org.springframework.data.annotation.Id;
import org.springframework.data.jdbc.core.mapping.AggregateReference;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.MappedCollection;
import org.springframework.data.relational.core.mapping.Table;

import java.sql.Date;
import java.sql.Timestamp;
import java.util.Set;

@Builder
@Table(value = "auth\".\"profiles")
public record Profile(
        @Id @Column("profile_id") Long id,
        String firstName,
        String middleName,
        String lastName,
        Date birthdate,
        @Column("user_id") AggregateReference<User, Long> userId,
//        @Column("user_id") Long userId,
        Set<Address> addresses,
        @MappedCollection(idColumn = "profile_id", keyColumn = "profile_id") ProfileIcon icon,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
    public void addAddress(Address address) {
        addresses.add(address);
    }

    public void removeAddress(Address address) {
        addresses.remove(address);
    }
}
