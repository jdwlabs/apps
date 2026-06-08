package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.models.User;
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
public class UserDaoPostgres implements UserDao {
    private final JdbcClient jdbcClient;

    @Override
    public User create(User user) {
        log.debug("Creating user: {}", user);
        String sql = """
                INSERT INTO auth.users (email_address, password, status, created_by_user_id, created_time, modified_by_user_id, modified_time)
                VALUES (:emailAddress, :password, :status, :createdByUserId, :createdTime, :modifiedByUserId, :modifiedTime)
                RETURNING user_id
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
        log.debug("Find user with id: {}", id);
        String sql = "SELECT * FROM auth.users WHERE user_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(this::userRowMapper)
                .optional();
    }

    @Override
    public Optional<User> findByEmailAddress(String emailAddress) {
        log.debug("Find user with email address: {}", emailAddress);
        String sql = "SELECT * FROM auth.users WHERE email_address = :emailAddress";
        return jdbcClient.sql(sql)
                .param("emailAddress", emailAddress)
                .query(this::userRowMapper)
                .optional();
    }

    @Override
    public List<User> findAll() {
        log.debug("Find all users");
        String sql = "SELECT * FROM auth.users";
        return jdbcClient.sql(sql)
                .query(this::userRowMapper)
                .list();
    }

    @Override
    public User update(User user) {
        log.debug("Updating user: {}", user);
        String sql = """
                UPDATE auth.users
                SET email_address       = :emailAddress,
                    password            = :password,
                    status              = :status,
                    modified_by_user_id = :modifiedByUserId,
                    modified_time       = :modifiedTime
                WHERE user_id = :id
                """;
        jdbcClient.sql(sql)
                .param("emailAddress", user.emailAddress())
                .param("password", user.password())
                .param("status", getStatus(user))
                .param("modifiedByUserId", user.modifiedByUserId())
                .param("modifiedTime", Timestamp.from(Instant.now()))
                .param("id", user.id())
                .update();
        return findById(user.id()).orElse(null);
    }

    @Override
    public void deleteById(Long id) {
        log.debug("Deleting user with id: {}", id);
        String sql = "DELETE FROM auth.users WHERE user_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    String getStatus(User user) {
        log.debug("Get user status: {}", user);
        return user.status() != null ? user.status() : Status.ACTIVE.name();
    }

    User userRowMapper(ResultSet rs, int rowNum) throws SQLException {
        log.debug("Mapping user: rs={}, rowNum={}", rs, rowNum);
        Long id = rs.getLong("user_id");
        String emailAddress = rs.getString("email_address");
        String password = rs.getString("password");
        String status = rs.getString("status");
        Long createdByUserId = rs.getLong("created_by_user_id");
        Timestamp createdTime = rs.getTimestamp("created_time");
        Long modifiedByUserId = rs.getLong("modified_by_user_id");
        Timestamp modifiedTime = rs.getTimestamp("modified_time");
        Set<UserRole> roles = new HashSet<>();
        return User.builder()
                .id(id)
                .emailAddress(emailAddress)
                .password(password)
                .status(status)
                .roles(roles)
                .profile(null)
                .createdByUserId(createdByUserId)
                .createdTime(createdTime)
                .modifiedByUserId(modifiedByUserId)
                .modifiedTime(modifiedTime)
                .build();
    }
}
