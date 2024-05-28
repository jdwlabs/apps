package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.models.User;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

import java.sql.Time;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;
import java.util.Optional;

@Repository
@RequiredArgsConstructor
@Slf4j
public class UserDaoPostgres implements UserDao {
    private final JdbcClient jdbcClient;

    @Override
    public User create(User user) {
        log.debug("Creating user: {}", user);
        String sql = """
                INSERT INTO auth.users (email_address, password, status, created_by_user_id, created_time, modified_by_user_id, modified_time)
                VALUES (:emailAddress, :password, :status, :createdByUserId, :createdTime, :modifiedByUserId, :modifiedTime)
                RETURNING auth.users.user_id
                """;
        Timestamp currentTime = Timestamp.from(Instant.now());
        Long generatedId = jdbcClient.sql(sql)
                .param("emailAddress", user.emailAddress())
                .param("password", user.password())
                .param("status", getStatus(user))
                .param("createdByUserId", user.createdByUserId())
                .param("createdTime", currentTime)
                .param("modifiedByUserId", user.modifiedByUserId())
                .param("modifiedTime", currentTime)
                .query(Long.class)
                .single();
        return findById(generatedId).orElse(null);
    }

    @Override
    public Optional<User> findById(Long id) {
        return Optional.empty();
    }

    @Override
    public List<User> findAll() {
        return List.of();
    }

    @Override
    public User update(User user) {
        return null;
    }

    @Override
    public void deleteById(Long id) {

    }

    private String getStatus(User user) {
        log.debug("Get user status: {}", user);
        return user.status() != null ? user.status() : Status.ACTIVE.name();
    }
}
