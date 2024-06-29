package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Address;
import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.models.ProfileIcon;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

import java.sql.*;
import java.time.Instant;
import java.util.HashSet;
import java.util.List;
import java.util.Optional;
import java.util.Set;

@Repository
@RequiredArgsConstructor
@Slf4j
public class ProfileDaoPostgres implements ProfileDao {
    private final JdbcClient jdbcClient;

    @Override
    public Profile create(Profile profile) {
        log.debug("Creating profile: {}", profile);
        String sql = """
                INSERT INTO auth.profiles (user_id, first_name, middle_name, last_name, birthdate, created_by_user_id, created_time, modified_by_user_id, modified_time)
                VALUES (:userId, :firstName, :middleName, :lastName, :birthdate, :createdByUserId, :createdTime, :modifiedByUserId, :modifiedTime)
                RETURNING profile_id
                """;
        Timestamp currentTime = Timestamp.from(Instant.now());
        Long generatedId = jdbcClient.sql(sql)
                .param("userId", profile.userId())
                .param("firstName", profile.firstName())
                .param("middleName", profile.middleName())
                .param("lastName", profile.lastName())
                .param("birthdate", profile.birthdate())
                .param("createdByUserId", profile.createdByUserId())
                .param("createdTime", currentTime)
                .param("modifiedByUserId", profile.modifiedByUserId())
                .param("modifiedTime", currentTime)
                .query(Long.class)
                .single();
        return findById(generatedId).orElse(null);
    }

    @Override
    public Optional<Profile> findById(Long id) {
        log.debug("Find profile with id: {}", id);
        String sql = "SELECT * FROM auth.profiles WHERE profile_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(this::profileRowMapper)
                .optional();
    }

    @Override
    public Optional<Profile> findByUserId(Long id) {
        log.debug("Find profile with user id: {}", id);
        String sql = "SELECT * FROM auth.profiles WHERE user_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(this::profileRowMapper)
                .optional();
    }

    @Override
    public List<Profile> findAll() {
        log.debug("Find all profiles");
        String sql = "SELECT * FROM auth.profiles";
        return jdbcClient.sql(sql)
                .query(this::profileRowMapper)
                .list();
    }

    @Override
    public Profile update(Profile profile) {
        log.debug("Updating profile: {}", profile);
        String sql = """
                UPDATE auth.profiles
                SET first_name          = :firstName,
                    middle_name         = :middleName,
                    last_name           = :lastName,
                    birthdate           = :birthdate,
                    modified_by_user_id = :modifiedByUserId,
                    modified_time       = :modifiedTime
                WHERE profile_id = :id
                """;
        jdbcClient.sql(sql)
                .param("firstName", profile.firstName())
                .param("middleName", profile.middleName())
                .param("lastName", profile.lastName())
                .param("birthdate", profile.birthdate())
                .param("modifiedByUserId", profile.modifiedByUserId())
                .param("modifiedTime", Timestamp.from(Instant.now()))
                .param("id", profile.id())
                .update();
        return findById(profile.id()).orElse(null);
    }

    @Override
    public void deleteById(Long id) {
        log.debug("Delete profile with id: {}", id);
        String sql = "DELETE FROM auth.profiles WHERE profile_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    @Override
    public void deleteByUserId(Long id) {
        log.debug("Delete profile with user id: {}", id);
        String sql = "DELETE FROM auth.profiles WHERE user_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    private Profile profileRowMapper(ResultSet rs, int rowNum) throws SQLException {
        log.debug("Mapping profile: rs={}, rowNum={}", rs, rowNum);
        Long id = rs.getLong("profile_id");
        Long userId = rs.getLong("user_id");
        String firstName = rs.getString("first_name");
        String middleName = rs.getString("middle_name");
        String lastName = rs.getString("last_name");
        Date birthdate = rs.getDate("birthdate");
        Long createdByUserId = rs.getLong("created_by_user_id");
        Timestamp createdTime = rs.getTimestamp("created_time");
        Long modifiedByUserId = rs.getLong("modified_by_user_id");
        Timestamp modifiedTime = rs.getTimestamp("modified_time");
        Set<Address> addresses = new HashSet<>();
        ProfileIcon profileIcon = null;
        return Profile.builder()
                .id(id)
                .firstName(firstName)
                .middleName(middleName)
                .lastName(lastName)
                .birthdate(birthdate)
                .userId(userId)
                .addresses(addresses)
                .icon(profileIcon)
                .createdByUserId(createdByUserId)
                .createdTime(createdTime)
                .modifiedByUserId(modifiedByUserId)
                .modifiedTime(modifiedTime)
                .build();
    }
}
