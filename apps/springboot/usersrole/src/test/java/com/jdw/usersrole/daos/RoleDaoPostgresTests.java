package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.Status;
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
import java.util.HashSet;
import java.util.List;
import java.util.Optional;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class RoleDaoPostgresTests {
    @Mock
    private JdbcClient jdbcClient;
    @InjectMocks
    private RoleDaoPostgres roleDaoPostgres;

    private Role buildMockRole() {
        Set<UserRole> mockUserRoles = new HashSet<>();
        return Role.builder()
                .id(1L)
                .name("ROLE_ADMIN")
                .description("Administrator role")
                .status(Status.ACTIVE.name())
                .users(mockUserRoles)
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void create_shouldReturnCreatedRole() {
        Role mockRole = buildMockRole();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Long> longMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        JdbcClient.MappedQuerySpec<Role> roleMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Long.class)).thenReturn(longMappedQuerySpec);
        when(longMappedQuerySpec.single()).thenReturn(1L);
        when(statementSpec.query(Mockito.<RowMapper<Role>>any())).thenReturn(roleMappedQuerySpec);
        when(roleMappedQuerySpec.optional()).thenReturn(Optional.of(mockRole));

        Role createdRole = roleDaoPostgres.create(mockRole);

        assertEquals(mockRole, createdRole);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void findById_shouldReturnRole() {
        Role mockRole = buildMockRole();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Role> roleMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Role>>any())).thenReturn(roleMappedQuerySpec);
        when(roleMappedQuerySpec.optional()).thenReturn(Optional.of(mockRole));

        Optional<Role> result = roleDaoPostgres.findById(1L);

        assertTrue(result.isPresent());
        assertEquals(mockRole, result.get());
    }

    @Test
    void findByName_shouldReturnRole() {
        Role mockRole = buildMockRole();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Role> roleMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Role>>any())).thenReturn(roleMappedQuerySpec);
        when(roleMappedQuerySpec.optional()).thenReturn(Optional.of(mockRole));

        Optional<Role> result = roleDaoPostgres.findByName("ROLE_ADMIN");

        assertTrue(result.isPresent());
        assertEquals(mockRole, result.get());
    }

    @Test
    void findAll_shouldReturnListOfRoles() {
        List<Role> mockRoles = List.of(buildMockRole());
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Role> roleMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Role>>any())).thenReturn(roleMappedQuerySpec);
        when(roleMappedQuerySpec.list()).thenReturn(mockRoles);

        List<Role> result = roleDaoPostgres.findAll();

        assertEquals(1, result.size());
        assertEquals(mockRoles.getFirst(), result.getFirst());
    }

    @Test
    void update_shouldReturnUpdatedRole() {
        Role mockRole = buildMockRole();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Role> roleMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Role>>any())).thenReturn(roleMappedQuerySpec);
        when(roleMappedQuerySpec.optional()).thenReturn(Optional.of(mockRole));

        Role updatedRole = roleDaoPostgres.update(mockRole);

        assertEquals(mockRole, updatedRole);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void deleteById_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        roleDaoPostgres.deleteById(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void getStatus_ifNoValue_shouldReturnActive() {
        String response = roleDaoPostgres.getStatus(Role.builder().build());

        assertEquals(Status.ACTIVE.name(), response);
    }

    @Test
    void getStatus_ifValue_shouldReturnRoleStatus() {
        String response = roleDaoPostgres.getStatus(Role.builder().status(Status.ACTIVE.name()).build());

        assertEquals(Status.ACTIVE.name(), response);
    }

    @Test
    void roleRowMapper_shouldMapResultSetToRole() throws SQLException {
        ResultSet resultSet = mock(ResultSet.class);
        when(resultSet.getLong("role_id")).thenReturn(1L);
        when(resultSet.getString("role_name")).thenReturn("ROLE_ADMIN");
        when(resultSet.getString("role_description")).thenReturn("Administrator role");
        when(resultSet.getString("status")).thenReturn("ACTIVE");
        when(resultSet.getLong("created_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("created_time")).thenReturn(Timestamp.from(Instant.now()));
        when(resultSet.getLong("modified_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("modified_time")).thenReturn(Timestamp.from(Instant.now()));

        Role result = roleDaoPostgres.roleRowMapper(resultSet, 0);

        assertEquals("ROLE_ADMIN", result.name());
        assertEquals("Administrator role", result.description());
        assertEquals("ACTIVE", result.status());
    }
}