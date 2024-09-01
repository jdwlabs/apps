package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.UserRole;
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

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class UserRoleDaoPostgresTests {
    @Mock
    private JdbcClient jdbcClient;
    @InjectMocks
    private UserRoleDaoPostgres userRoleDaoPostgres;

    private UserRole buildMockUserRole() {
        return UserRole.builder()
                .userId(1L)
                .roleId(1L)
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void create_shouldReturnCreatedUserRole() {
        UserRole mockUserRole = buildMockUserRole();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        UserRole createdUserRole = userRoleDaoPostgres.create(mockUserRole);

        assertEquals(mockUserRole.userId(), createdUserRole.userId());
        assertEquals(mockUserRole.roleId(), createdUserRole.roleId());
        verify(jdbcClient, times(1)).sql(any(String.class));
    }

    @Test
    void findByRoleId_shouldReturnListOfUserRoles() {
        List<UserRole> mockUserRoles = List.of(buildMockUserRole());
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<UserRole> userRoleMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<UserRole>>any())).thenReturn(userRoleMappedQuerySpec);
        when(userRoleMappedQuerySpec.list()).thenReturn(mockUserRoles);

        List<UserRole> result = userRoleDaoPostgres.findByRoleId(1L);

        assertEquals(1, result.size());
        assertEquals(mockUserRoles.getFirst(), result.getFirst());
    }

    @Test
    void findByUserId_shouldReturnListOfUserRoles() {
        List<UserRole> mockUserRoles = List.of(buildMockUserRole());
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<UserRole> userRoleMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<UserRole>>any())).thenReturn(userRoleMappedQuerySpec);
        when(userRoleMappedQuerySpec.list()).thenReturn(mockUserRoles);

        List<UserRole> result = userRoleDaoPostgres.findByUserId(1L);

        assertEquals(1, result.size());
        assertEquals(mockUserRoles.getFirst(), result.getFirst());
    }

    @Test
    void deleteByRoleId_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        userRoleDaoPostgres.deleteByRoleId(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void deleteByUserId_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        userRoleDaoPostgres.deleteByUserId(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void userRoleRowMapper_shouldMapResultSetToUserRole() throws SQLException {
        ResultSet resultSet = mock(ResultSet.class);
        when(resultSet.getLong("user_id")).thenReturn(1L);
        when(resultSet.getLong("role_id")).thenReturn(1L);
        when(resultSet.getLong("created_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("created_time")).thenReturn(Timestamp.from(Instant.now()));

        UserRole result = userRoleDaoPostgres.userRoleRowMapper(resultSet, 0);

        assertEquals(1L, result.userId());
        assertEquals(1L, result.roleId());
        assertEquals(1L, result.createdByUserId());
    }
}