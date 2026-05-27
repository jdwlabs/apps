package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.UserRole;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;
import java.util.Optional;

@Repository
@RequiredArgsConstructor
@Slf4j
public class UserRoleDaoPostgres implements UserRoleDao {
    private final JdbcClient jdbcClient;

    @Override
    public UserRole create(UserRole userRole) {
        log.debug("Creating user role: {}", userRole);
        String sql = """
                INSERT INTO auth.users_roles (user_id, role_id, created_by_user_id, created_time)
                VALUES (:userId, :roleId, :createdByUserId, :createdTime)
                """;
        Timestamp currentTime = Timestamp.from(Instant.now());
        jdbcClient.sql(sql)
                .param("userId", userRole.userId())
                .param("roleId", userRole.roleId())
                .param("createdByUserId", userRole.createdByUserId())
                .param("createdTime", currentTime)
                .update();
        return UserRole.builder()
                .userId(userRole.userId())
                .roleId(userRole.roleId())
                .createdByUserId(userRole.createdByUserId())
                .createdTime(currentTime)
                .build();
    }

    @Override
    public List<UserRole> findByRoleId(Long roleId) {
        log.debug("Finding user role with role id: {}", roleId);
        String sql = "SELECT * FROM auth.users_roles WHERE role_id = :roleId";
        return jdbcClient.sql(sql)
                .param("roleId", roleId)
                .query(this::userRoleRowMapper)
                .list();
    }

    @Override
    public List<UserRole> findByUserId(Long userId) {
        log.debug("Finding user role with user id: {}", userId);
        String sql = "SELECT * FROM auth.users_roles WHERE user_id = :userId";
        return jdbcClient.sql(sql)
                .param("userId", userId)
                .query(this::userRoleRowMapper)
                .list();
    }

    @Override
    public Optional<UserRole> findByRoleIdAndUserId(Long roleId, Long userId) {
        log.debug("Finding user role with role id: {} and user id: {}", roleId, userId);
        String sql = """
            SELECT * FROM auth.users_roles
            WHERE role_id = :roleId AND user_id = :userId
            """;
        return jdbcClient.sql(sql)
                .param("roleId", roleId)
                .param("userId", userId)
                .query(this::userRoleRowMapper)
                .optional();
    }

    @Override
    public void deleteByRoleId(Long roleId) {
        log.debug("Deleting user role with role id: {}", roleId);
        String sql = "DELETE FROM auth.users_roles WHERE role_id = :roleId";
        jdbcClient.sql(sql)
                .param("roleId", roleId)
                .update();
    }

    @Override
    public void deleteByUserId(Long userId) {
        log.debug("Deleting user role with user id: {}", userId);
        String sql = "DELETE FROM auth.users_roles WHERE user_id = :userId";
        jdbcClient.sql(sql)
                .param("userId", userId)
                .update();
    }

    @Override
    public void deleteByRoleIdAndUserId(Long roleId, Long userId) {
        log.debug("Deleting user role with role id: {} and user id: {}", roleId, userId);
        String sql = """
            DELETE FROM auth.users_roles
            WHERE role_id = :roleId AND user_id = :userId
            """;
        jdbcClient.sql(sql)
                .param("roleId", roleId)
                .param("userId", userId)
                .update();
    }

    UserRole userRoleRowMapper(ResultSet rs, int rowNum) throws SQLException {
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
