package com.jdw.usersrole.daos;

import com.jdw.usersrole.daos.rowmappers.RoleRowMapper;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.Status;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;
import java.util.Optional;

@Repository
@RequiredArgsConstructor
@Slf4j
public class RoleDaoPostgres implements RoleDao {
    private final JdbcClient jdbcClient;

    @Override
    public Role create(Role role) {
        log.debug("Create role: {}", role);
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
        log.debug("Find role by id: {}", id);
        String sql = "SELECT * FROM auth.roles WHERE role_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(new RoleRowMapper())
                .optional();
    }

    @Override
    public Optional<Role> findByName(String name) {
        log.debug("Find role by name: {}", name);
        String sql = "SELECT * FROM auth.roles WHERE role_name = :name";
        return jdbcClient.sql(sql)
                .param("name", name)
                .query(new RoleRowMapper())
                .optional();
    }

    @Override
    public List<Role> findAll() {
        log.debug("Find all roles");
        String sql = "SELECT * FROM auth.roles";
        return jdbcClient.sql(sql)
                .query(new RoleRowMapper())
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
                WHERE role_id = :roleId
                """;
        jdbcClient.sql(sql)
                .param("name", role.name())
                .param("description", role.description())
                .param("status", getStatus(role))
                .param("modifiedByUserId", role.modifiedByUserId())
                .param("modifiedTime", Timestamp.from(Instant.now()))
                .param("roleId", role.id())
                .update();
        return findById(role.id()).orElse(null);
    }

    @Override
    public void deleteById(Long id) {
        log.debug("Delete role by id: {}", id);
        String sql = "DELETE FROM auth.roles WHERE role_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    private String getStatus(Role role) {
        log.debug("Get role status: {}", role);
        return role.status() != null ? role.status() : Status.ACTIVE.name();
    }
}
