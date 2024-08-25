package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Address;
import com.jdw.usersrole.models.Profile;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.Mockito;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.simple.JdbcClient;

import java.sql.Date;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.HashSet;
import java.util.List;
import java.util.Optional;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class ProfileDaoPostgresTests {
    @Mock
    private JdbcClient jdbcClient;
    @InjectMocks
    private ProfileDaoPostgres profileDaoPostgres;

    private Profile buildMockProfile() {
        Set<Address> mockAddresses = new HashSet<>();
        return Profile.builder()
                .id(1L)
                .userId(1L)
                .firstName("John")
                .middleName("M")
                .lastName("Doe")
                .birthdate(Date.valueOf("1990-01-01"))
                .addresses(mockAddresses)
                .icon(null)
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void create_shouldReturnCreatedProfile() {
        Profile mockProfile = buildMockProfile();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Long> longMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        JdbcClient.MappedQuerySpec<Profile> profileMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Long.class)).thenReturn(longMappedQuerySpec);
        when(longMappedQuerySpec.single()).thenReturn(1L);
        when(statementSpec.query(Mockito.<RowMapper<Profile>>any())).thenReturn(profileMappedQuerySpec);
        when(profileMappedQuerySpec.optional()).thenReturn(Optional.of(mockProfile));

        Profile createdProfile = profileDaoPostgres.create(mockProfile);

        assertEquals(mockProfile, createdProfile);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void findById_shouldReturnProfile() {
        Profile mockProfile = buildMockProfile();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Profile> profileMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Profile>>any())).thenReturn(profileMappedQuerySpec);
        when(profileMappedQuerySpec.optional()).thenReturn(Optional.of(mockProfile));

        Optional<Profile> result = profileDaoPostgres.findById(1L);

        assertTrue(result.isPresent());
        assertEquals(mockProfile, result.get());
    }

    @Test
    void findByUserId_shouldReturnProfile() {
        Profile mockProfile = buildMockProfile();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Profile> profileMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Profile>>any())).thenReturn(profileMappedQuerySpec);
        when(profileMappedQuerySpec.optional()).thenReturn(Optional.of(mockProfile));

        Optional<Profile> result = profileDaoPostgres.findByUserId(1L);

        assertTrue(result.isPresent());
        assertEquals(mockProfile, result.get());
    }

    @Test
    void findAll_shouldReturnListOfProfiles() {
        List<Profile> mockProfiles = List.of(buildMockProfile());
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Profile> profileMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Profile>>any())).thenReturn(profileMappedQuerySpec);
        when(profileMappedQuerySpec.list()).thenReturn(mockProfiles);

        List<Profile> result = profileDaoPostgres.findAll();

        assertEquals(1, result.size());
        assertEquals(mockProfiles.getFirst(), result.getFirst());
    }

    @Test
    void update_shouldReturnUpdatedProfile() {
        Profile mockProfile = buildMockProfile();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Profile> profileMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Profile>>any())).thenReturn(profileMappedQuerySpec);
        when(profileMappedQuerySpec.optional()).thenReturn(Optional.of(mockProfile));

        Profile updatedProfile = profileDaoPostgres.update(mockProfile);

        assertEquals(mockProfile, updatedProfile);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void deleteById_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        profileDaoPostgres.deleteById(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void deleteByUserId_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        profileDaoPostgres.deleteByUserId(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void profileRowMapper_shouldMapResultSetToProfile() throws SQLException {
        ResultSet resultSet = mock(ResultSet.class);
        when(resultSet.getLong("profile_id")).thenReturn(1L);
        when(resultSet.getLong("user_id")).thenReturn(1L);
        when(resultSet.getString("first_name")).thenReturn("John");
        when(resultSet.getString("middle_name")).thenReturn("M");
        when(resultSet.getString("last_name")).thenReturn("Doe");
        when(resultSet.getDate("birthdate")).thenReturn(Date.valueOf("1990-01-01"));
        when(resultSet.getLong("created_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("created_time")).thenReturn(Timestamp.from(Instant.now()));
        when(resultSet.getLong("modified_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("modified_time")).thenReturn(Timestamp.from(Instant.now()));

        Profile result = profileDaoPostgres.profileRowMapper(resultSet, 0);

        assertEquals("John", result.firstName());
        assertEquals("M", result.middleName());
        assertEquals("Doe", result.lastName());
        assertEquals(Date.valueOf("1990-01-01"), result.birthdate());
    }
}