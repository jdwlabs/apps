package com.jdw.usersrole.daos.rowmappers;

import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.UserRole;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.RowMapper;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.util.HashSet;
import java.util.Set;

@Slf4j
public class RoleRowMapper implements RowMapper<Role> {
    @Override
    public Role mapRow(ResultSet rs, int rowNum) throws SQLException {
        log.debug("Mapping role: rs={}, rowNum={}", rs, rowNum);
        Long id = rs.getLong("role_id");
        String name = rs.getString("role_name");
        String description = rs.getString("role_description");
        String status = rs.getString("status");
        Long createdByUserId = rs.getLong("created_by_user_id");
        Timestamp createdTime = rs.getTimestamp("created_time");
        Long modifiedByUserId = rs.getLong("modified_by_user_id");
        Timestamp modifiedTime = rs.getTimestamp("modified_time");
        Set<UserRole> users = new HashSet<>();
        return Role.builder()
                .id(id)
                .name(name)
                .description(description)
                .status(status)
                .users(users)
                .createdByUserId(createdByUserId)
                .createdTime(createdTime)
                .modifiedByUserId(modifiedByUserId)
                .modifiedTime(modifiedTime)
                .build();
    }
}
