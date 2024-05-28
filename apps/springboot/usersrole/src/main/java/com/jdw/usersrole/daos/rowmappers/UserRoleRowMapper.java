package com.jdw.usersrole.daos.rowmappers;

import com.jdw.usersrole.models.UserRole;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.RowMapper;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;

@Slf4j
public class UserRoleRowMapper implements RowMapper<UserRole> {
    @Override
    public UserRole mapRow(ResultSet rs, int rowNum) throws SQLException {
        log.debug("Mapping user role: rs={}, rowNum={}", rs, rowNum);
        Long userId = rs.getLong("user_id");
        Long roleId = rs.getLong("role_id");
        Long createdByUserId = rs.getLong("created_by_user_id");
        Timestamp createdTime = rs.getTimestamp("created_time");
        return UserRole.builder()
                .userId(userId)
                .roleId(roleId)
                .createdByUserId(createdByUserId)
                .createdTime(createdTime)
                .build();
    }
}
