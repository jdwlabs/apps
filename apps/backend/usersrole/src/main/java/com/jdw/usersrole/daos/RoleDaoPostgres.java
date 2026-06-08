package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.models.UserRole;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.HashSet;
import java.util.List;
import java.util.Optional;
import java.util.Set;

@Repository
@RequiredArgsConstructor
@Slf4j
public class RoleDaoPostgres implements RoleDao {
    private final JdbcClient jdbcClient;

    @Override
    public Role create(Role role) {
        log.debug("Creating role: {}", role);
        String sql = """
                INSERT INTO auth.roles (role_name, role_description, status, created_by_user_id, created_time,
                        modified_by_user_id, modified_time)
                VALUES (:name, :description, :status, :createdByUserId, :createdTime, :modifiedByUserId, :modifiedTime)
                RETURNING auth.roles.role_id
                """;
        Timestamp currentTime = Timestamp.from(Instant.now());
        Long generatedId = jdbcClient.sql(sql)
                .param("name", role.name())
                .param("description", role.description())
                .param("status", getStatus(role))
                .param("createdByUserId", role.createdByUserId())
                .param("createdTime", currentTime)
                .param("modifiedByUserId", role.modifiedByUserId())
                .param("modifiedTime", currentTime)
                .query(Long.class)
                .single();
        return findById(generatedId).orElse(null);
    }

    @Override
    public Optional<Role> findById(Long id) {
        log.debug("Find role with id: {}", id);
        String sql = "SELECT * FROM auth.roles WHERE role_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(this::roleRowMapper)
                .optional();
    }

    @Override
    public Optional<Role> findByName(String name) {
        log.debug("Find role with name: {}", name);
        String sql = "SELECT * FROM auth.roles WHERE role_name = :name";
        return jdbcClient.sql(sql)
                .param("name", name)
                .query(this::roleRowMapper)
                .optional();
    }

    @Override
    public List<Role> findAll() {
        log.debug("Find all roles");
        String sql = "SELECT * FROM auth.roles";
        return jdbcClient.sql(sql)
                .query(this::roleRowMapper)
                .list();
    }

    @Override
    public Role update(Role role) {
        log.debug("Update role: {}", role);
        String sql = """
                UPDATE auth.roles
                SET role_name           = :name,
                    role_description    = :description,
                    status              = :status,
                    modified_by_user_id = :modifiedByUserId,
                    modified_time       = :modifiedTime
                WHERE role_id = :id
                """;
        jdbcClient.sql(sql)
                .param("name", role.name())
                .param("description", role.description())
                .param("status", getStatus(role))
                .param("modifiedByUserId", role.modifiedByUserId())
                .param("modifiedTime", Timestamp.from(Instant.now()))
                .param("id", role.id())
                .update();
        return findById(role.id()).orElse(null);
    }

    @Override
    public void deleteById(Long id) {
        log.debug("Delete role with id: {}", id);
        String sql = "DELETE FROM auth.roles WHERE role_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    String getStatus(Role role) {
        log.debug("Get role status: {}", role);
        return role.status() != null ? role.status() : Status.ACTIVE.name();
    }

    Role roleRowMapper(ResultSet rs, int rowNum) throws SQLException {
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
