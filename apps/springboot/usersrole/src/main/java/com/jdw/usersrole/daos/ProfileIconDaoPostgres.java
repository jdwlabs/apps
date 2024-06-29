package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.ProfileIcon;
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
public class ProfileIconDaoPostgres implements ProfileIconDao {
    private final JdbcClient jdbcClient;

    @Override
    public ProfileIcon create(ProfileIcon profileIcon) {
        log.debug("Creating profile icon: {}", profileIcon);
        String sql = """
                INSERT INTO auth.profile_icons (profile_id, icon, created_by_user_id, created_time, modified_by_user_id, modified_time)
                VALUES (:profileId, :icon, :createdByUserId, :createdTime, :modifiedByUserId, :modifiedTime)
                RETURNING profile_id
                """;
        Timestamp currentTime = Timestamp.from(Instant.now());
        Long generatedId = jdbcClient.sql(sql)
                .param("profileId", profileIcon.profileId())
                .param("icon", profileIcon.icon())
                .param("createdByUserId", profileIcon.createdByUserId())
                .param("createdTime", currentTime)
                .param("modifiedByUserId", profileIcon.modifiedByUserId())
                .param("modifiedTime", currentTime)
                .query(Long.class)
                .single();
        return findById(generatedId).orElse(null);
    }

    @Override
    public Optional<ProfileIcon> findById(Long id) {
        log.debug("Find profile icon with id: {}", id);
        String sql = "SELECT * FROM auth.profile_icons WHERE profile_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(this::profileIconRowMapper)
                .optional();
    }

    @Override
    public Optional<ProfileIcon> findByProfileId(Long id) {
        log.debug("Find profile icon with profile id: {}", id);
        String sql = "SELECT * FROM auth.profile_icons WHERE profile_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(this::profileIconRowMapper)
                .optional();
    }

    @Override
    public List<ProfileIcon> findAll() {
        log.debug("Find all profile icons");
        String sql = "SELECT * FROM auth.profile_icons";
        return jdbcClient.sql(sql)
                .query(this::profileIconRowMapper)
                .list();
    }

    @Override
    public ProfileIcon update(ProfileIcon profileIcon) {
        log.debug("Update profile icon: {}", profileIcon);
        String sql = """
                UPDATE auth.profile_icons
                SET icon                = :icon,
                    modified_by_user_id = :modifiedByUserId,
                    modified_time       = :modifiedTime
                WHERE icon_id = :id
                """;
        jdbcClient.sql(sql)
                .param("icon", profileIcon.icon())
                .param("modifiedByUserId", profileIcon.createdByUserId())
                .param("modifiedTime", profileIcon.modifiedTime())
                .param("id", profileIcon.id())
                .update();
        return findByProfileId(profileIcon.id()).orElse(null);
    }

    @Override
    public void deleteById(Long id) {
        log.debug("Delete profile icon with id: {}", id);
        String sql = "DELETE FROM auth.profile_icons WHERE icon_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    @Override
    public void deleteByProfileId(Long id) {
        log.debug("Delete profile icon with profile id: {}", id);
        String sql = "DELETE FROM auth.profile_icons WHERE profile_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    private ProfileIcon profileIconRowMapper(ResultSet rs, int rowNum) throws SQLException {
        log.debug("Mapping profile icon: rs={}, rowNum={}", rs, rowNum);
        Long id = rs.getLong("icon_id");
        Long profileId = rs.getLong("profile_id");
        byte[] icon = rs.getBytes("icon");
        Long createdByUserId = rs.getLong("created_by_user_id");
        Timestamp createdTime = rs.getTimestamp("created_time");
        Long modifiedByUserId = rs.getLong("modified_by_user_id");
        Timestamp modifiedTime = rs.getTimestamp("modified_time");
        return ProfileIcon.builder()
                .id(id)
                .profileId(profileId)
                .icon(icon)
                .createdByUserId(createdByUserId)
                .createdTime(createdTime)
                .modifiedByUserId(modifiedByUserId)
                .modifiedTime(modifiedTime)
                .build();
    }
}
