package com.jdw.usersrole.daos;

import com.jdw.usersrole.daos.rowmappers.UserRoleRowMapper;
import com.jdw.usersrole.models.UserRole;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;

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
        log.debug("Finding user role by role id: {}", roleId);
        String sql = "SELECT * FROM auth.users_roles WHERE role_id = :roleId";
        return jdbcClient.sql(sql)
                .param("roleId", roleId)
                .query(new UserRoleRowMapper())
                .list();
    }

    @Override
    public List<UserRole> findByUserId(Long userId) {
        log.debug("Finding user role by user id: {}", userId);
        String sql = "SELECT * FROM auth.users_roles WHERE user_id = :userId";
        return jdbcClient.sql(sql)
                .param("userId", userId)
                .query(new UserRoleRowMapper())
                .list();
    }

    @Override
    public void deleteByRoleId(Long roleId) {
        log.debug("Deleting user role by role id: {}", roleId);
        String sql = "DELETE FROM auth.users_roles WHERE role_id = :roleId";
        jdbcClient.sql(sql)
                .param("roleId", roleId)
                .update();
    }

    @Override
    public void deleteByUserId(Long userId) {
        log.debug("Deleting user role by user id: {}", userId);
        String sql = "DELETE FROM auth.users_roles WHERE user_id = :userId";
        jdbcClient.sql(sql)
                .param("userId", userId)
                .update();

    }
}
