package com.jdw.usersrole.models;

import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Builder;

import java.sql.Timestamp;
import java.util.Set;

@Builder
public record User(
        Long id,
        String emailAddress,
        @JsonIgnore
        String password,
        String status,
        Set<UserRole> roles,
        Profile profile,
        Long createdByUserId,
        Timestamp createdTime,
        Long modifiedByUserId,
        Timestamp modifiedTime
) {
    // Same reasoning as UserRequestDTO: this holds the password hash, @JsonIgnore keeps it
    // out of JSON but does nothing for toString(), and a record's canonical toString()
    // includes every component regardless.
    @Override
    public String toString() {
        return "User[id=" + id +
                ", emailAddress=" + emailAddress +
                ", password=***" +
                ", status=" + status +
                ", roles=" + roles +
                ", profile=" + profile +
                ", createdByUserId=" + createdByUserId +
                ", createdTime=" + createdTime +
                ", modifiedByUserId=" + modifiedByUserId +
                ", modifiedTime=" + modifiedTime +
                "]";
    }
}
