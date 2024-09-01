package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.models.User;
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
class UserDaoPostgresTests {
    @Mock
    private JdbcClient jdbcClient;
    @InjectMocks
    private UserDaoPostgres userDaoPostgres;

    private User buildMockUser() {
        Set<UserRole> mockUserRoles = new HashSet<>();
        return User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .password("P@ssw0rd!")
                .status(Status.ACTIVE.name())
                .roles(mockUserRoles)
                .profile(null)
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void create_shouldReturnCreatedUser() {
        User mockUser = buildMockUser();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Long> longMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        JdbcClient.MappedQuerySpec<User> userMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Long.class)).thenReturn(longMappedQuerySpec);
        when(longMappedQuerySpec.single()).thenReturn(1L);
        when(statementSpec.query(Mockito.<RowMapper<User>>any())).thenReturn(userMappedQuerySpec);
        when(userMappedQuerySpec.optional()).thenReturn(Optional.of(mockUser));

        User createdUser = userDaoPostgres.create(mockUser);

        assertEquals(mockUser, createdUser);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void findById_shouldReturnUser() {
        User mockUser = buildMockUser();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<User> userMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<User>>any())).thenReturn(userMappedQuerySpec);
        when(userMappedQuerySpec.optional()).thenReturn(Optional.of(mockUser));

        Optional<User> result = userDaoPostgres.findById(1L);

        assertTrue(result.isPresent());
        assertEquals(mockUser, result.get());
    }

    @Test
    void findByEmailAddress_shouldReturnUser() {
        User mockUser = buildMockUser();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<User> userMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<User>>any())).thenReturn(userMappedQuerySpec);
        when(userMappedQuerySpec.optional()).thenReturn(Optional.of(mockUser));

        Optional<User> result = userDaoPostgres.findByEmailAddress("user@jdw.com");

        assertTrue(result.isPresent());
        assertEquals(mockUser, result.get());
    }

    @Test
    void findAll_shouldReturnListOfUsers() {
        List<User> mockUsers = List.of(buildMockUser());
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<User> userMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<User>>any())).thenReturn(userMappedQuerySpec);
        when(userMappedQuerySpec.list()).thenReturn(mockUsers);

        List<User> result = userDaoPostgres.findAll();

        assertEquals(1, result.size());
        assertEquals(mockUsers.getFirst(), result.getFirst());
    }

    @Test
    void update_shouldReturnUpdatedUser() {
        User mockUser = buildMockUser();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<User> userMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<User>>any())).thenReturn(userMappedQuerySpec);
        when(userMappedQuerySpec.optional()).thenReturn(Optional.of(mockUser));

        User updatedUser = userDaoPostgres.update(mockUser);

        assertEquals(mockUser, updatedUser);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void deleteById_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        userDaoPostgres.deleteById(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void getStatus_ifNoValue_shouldReturnActive() {
        String response = userDaoPostgres.getStatus(User.builder().build());

        assertEquals(Status.ACTIVE.name(), response);
    }

    @Test
    void getStatus_ifValue_shouldReturnRoleStatus() {
        String response = userDaoPostgres.getStatus(User.builder().status(Status.ACTIVE.name()).build());

        assertEquals(Status.ACTIVE.name(), response);
    }

    @Test
    void userRowMapper_shouldMapResultSetToUser() throws SQLException {
        ResultSet resultSet = mock(ResultSet.class);
        when(resultSet.getLong("user_id")).thenReturn(1L);
        when(resultSet.getString("email_address")).thenReturn("user@jdw.com");
        when(resultSet.getString("password")).thenReturn("P@ssw0rd!");
        when(resultSet.getString("status")).thenReturn(Status.ACTIVE.name());
        when(resultSet.getLong("created_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("created_time")).thenReturn(Timestamp.from(Instant.now()));
        when(resultSet.getLong("modified_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("modified_time")).thenReturn(Timestamp.from(Instant.now()));

        User result = userDaoPostgres.userRowMapper(resultSet, 0);

        assertEquals("user@jdw.com", result.emailAddress());
        assertEquals("P@ssw0rd!", result.password());
        assertEquals(Status.ACTIVE.name(), result.status());
    }
}