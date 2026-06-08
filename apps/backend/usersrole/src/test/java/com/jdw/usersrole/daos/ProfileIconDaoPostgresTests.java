package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.ProfileIcon;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.Mockito;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.simple.JdbcClient;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class ProfileIconDaoPostgresTests {
    @Mock
    private JdbcClient jdbcClient;
    @InjectMocks
    private ProfileIconDaoPostgres profileIconDaoPostgres;

    private ProfileIcon buildMockProfileIcon() {
        return ProfileIcon.builder()
                .id(1L)
                .profileId(1L)
                .icon(new byte[]{1, 2, 3})
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void create_shouldReturnCreatedProfileIcon() {
        ProfileIcon mockProfileIcon = buildMockProfileIcon();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Long> longMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        JdbcClient.MappedQuerySpec<ProfileIcon> profileIconMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Long.class)).thenReturn(longMappedQuerySpec);
        when(longMappedQuerySpec.single()).thenReturn(1L);
        when(statementSpec.query(Mockito.<RowMapper<ProfileIcon>>any())).thenReturn(profileIconMappedQuerySpec);
        when(profileIconMappedQuerySpec.optional()).thenReturn(Optional.of(mockProfileIcon));

        ProfileIcon createdProfileIcon = profileIconDaoPostgres.create(mockProfileIcon);

        assertEquals(mockProfileIcon, createdProfileIcon);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void findById_shouldReturnProfileIcon() {
        ProfileIcon mockProfileIcon = buildMockProfileIcon();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<ProfileIcon> profileIconMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<ProfileIcon>>any())).thenReturn(profileIconMappedQuerySpec);
        when(profileIconMappedQuerySpec.optional()).thenReturn(Optional.of(mockProfileIcon));

        Optional<ProfileIcon> result = profileIconDaoPostgres.findById(1L);

        assertTrue(result.isPresent());
        assertEquals(mockProfileIcon, result.get());
    }

    @Test
    void findByProfileId_shouldReturnProfileIcon() {
        ProfileIcon mockProfileIcon = buildMockProfileIcon();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<ProfileIcon> profileIconMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<ProfileIcon>>any())).thenReturn(profileIconMappedQuerySpec);
        when(profileIconMappedQuerySpec.optional()).thenReturn(Optional.of(mockProfileIcon));

        Optional<ProfileIcon> result = profileIconDaoPostgres.findByProfileId(1L);

        assertTrue(result.isPresent());
        assertEquals(mockProfileIcon, result.get());
    }

    @Test
    void findAll_shouldReturnListOfProfileIcons() {
        List<ProfileIcon> mockProfileIcons = List.of(buildMockProfileIcon());
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<ProfileIcon> profileIconMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<ProfileIcon>>any())).thenReturn(profileIconMappedQuerySpec);
        when(profileIconMappedQuerySpec.list()).thenReturn(mockProfileIcons);

        List<ProfileIcon> result = profileIconDaoPostgres.findAll();

        assertEquals(1, result.size());
        assertEquals(mockProfileIcons.getFirst(), result.getFirst());
    }

    @Test
    void update_shouldReturnUpdatedProfileIcon() {
        ProfileIcon mockProfileIcon = buildMockProfileIcon();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<ProfileIcon> profileIconMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<ProfileIcon>>any())).thenReturn(profileIconMappedQuerySpec);
        when(profileIconMappedQuerySpec.optional()).thenReturn(Optional.of(mockProfileIcon));

        ProfileIcon updatedProfileIcon = profileIconDaoPostgres.update(mockProfileIcon);

        assertEquals(mockProfileIcon, updatedProfileIcon);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void deleteById_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        profileIconDaoPostgres.deleteById(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void deleteByProfileId_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        profileIconDaoPostgres.deleteByProfileId(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void profileIconRowMapper_shouldMapResultSetToProfileIcon() throws SQLException {
        byte[] icon = new byte[]{1, 2, 3};
        ResultSet resultSet = mock(ResultSet.class);
        when(resultSet.getLong("icon_id")).thenReturn(1L);
        when(resultSet.getLong("profile_id")).thenReturn(1L);
        when(resultSet.getBytes("icon")).thenReturn(icon);
        when(resultSet.getLong("created_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("created_time")).thenReturn(Timestamp.from(Instant.now()));
        when(resultSet.getLong("modified_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("modified_time")).thenReturn(Timestamp.from(Instant.now()));

        ProfileIcon result = profileIconDaoPostgres.profileIconRowMapper(resultSet, 0);

        assertEquals(1L, result.id());
        assertEquals(1L, result.profileId());
        assertEquals(1L, result.createdByUserId());
        assertEquals(1L, result.modifiedByUserId());
        assertEquals(icon, result.icon());
    }
}