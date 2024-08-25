package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Address;
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
class AddressDaoPostgresTests {
    @Mock
    private JdbcClient jdbcClient;
    @InjectMocks
    private AddressDaoPostgres addressDaoPostgres;

    private Address buildMockAddress() {
        return Address.builder()
                .id(1L)
                .profileId(1L)
                .addressLine1("123 Main St")
                .addressLine2("Apt 4B")
                .city("Cityville")
                .stateProvince("Stateville")
                .postalCode("12345")
                .country("Countryland")
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void create_shouldReturnCreatedAddress() {
        Address mockAddress = buildMockAddress();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Long> longMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        JdbcClient.MappedQuerySpec<Address> addressMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Long.class)).thenReturn(longMappedQuerySpec);
        when(longMappedQuerySpec.single()).thenReturn(1L);
        when(statementSpec.query(Mockito.<RowMapper<Address>>any())).thenReturn(addressMappedQuerySpec);
        when(addressMappedQuerySpec.optional()).thenReturn(Optional.of(mockAddress));

        Address createdAddress = addressDaoPostgres.create(mockAddress);

        assertEquals(mockAddress, createdAddress);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void findById_shouldReturnAddress() {
        Address mockAddress = buildMockAddress();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Address> addressMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Address>>any())).thenReturn(addressMappedQuerySpec);
        when(addressMappedQuerySpec.optional()).thenReturn(Optional.of(mockAddress));

        Optional<Address> result = addressDaoPostgres.findById(1L);

        assertTrue(result.isPresent());
        assertEquals(mockAddress, result.get());
    }

    @Test
    void findByProfileId_shouldReturnListOfAddresses() {
        List<Address> mockAddresses = List.of(buildMockAddress());
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Address> addressMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Address>>any())).thenReturn(addressMappedQuerySpec);
        when(addressMappedQuerySpec.list()).thenReturn(mockAddresses);

        List<Address> result = addressDaoPostgres.findByProfileId(1L);

        assertEquals(1, result.size());
        assertEquals(mockAddresses.getFirst(), result.getFirst());
    }

    @Test
    void findAll_shouldReturnListOfAddresses() {
        List<Address> mockAddresses = List.of(buildMockAddress());
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Address> addressMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Address>>any())).thenReturn(addressMappedQuerySpec);
        when(addressMappedQuerySpec.list()).thenReturn(mockAddresses);

        List<Address> result = addressDaoPostgres.findAll();

        assertEquals(1, result.size());
        assertEquals(mockAddresses.getFirst(), result.getFirst());
    }

    @Test
    void update_shouldReturnUpdatedAddress() {
        Address mockAddress = buildMockAddress();
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        JdbcClient.MappedQuerySpec<Address> addressMappedQuerySpec = mock(String.valueOf(JdbcClient.MappedQuerySpec.class));
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);
        when(statementSpec.query(Mockito.<RowMapper<Address>>any())).thenReturn(addressMappedQuerySpec);
        when(addressMappedQuerySpec.optional()).thenReturn(Optional.of(mockAddress));

        Address updatedAddress = addressDaoPostgres.update(mockAddress);

        assertEquals(mockAddress, updatedAddress);
        verify(jdbcClient, times(2)).sql(any(String.class));
    }

    @Test
    void deleteById_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        addressDaoPostgres.deleteById(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void deleteByProfileId_shouldInvokeDeleteOperation() {
        JdbcClient.StatementSpec statementSpec = mock(JdbcClient.StatementSpec.class);
        when(jdbcClient.sql(anyString())).thenReturn(statementSpec);
        when(statementSpec.param(anyString(), any())).thenReturn(statementSpec);

        addressDaoPostgres.deleteByProfileId(1L);

        verify(statementSpec, times(1)).update();
    }

    @Test
    void addressRowMapper_shouldMapResultSetToAddress() throws SQLException {
        ResultSet resultSet = mock(ResultSet.class);
        when(resultSet.getLong("address_id")).thenReturn(1L);
        when(resultSet.getLong("profile_id")).thenReturn(1L);
        when(resultSet.getString("address_line_1")).thenReturn("123 Main St");
        when(resultSet.getString("address_line_2")).thenReturn("Apt 4B");
        when(resultSet.getString("city")).thenReturn("Cityville");
        when(resultSet.getString("state_province")).thenReturn("Stateville");
        when(resultSet.getString("postal_code")).thenReturn("12345");
        when(resultSet.getString("country")).thenReturn("Countryland");
        when(resultSet.getLong("created_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("created_time")).thenReturn(Timestamp.from(Instant.now()));
        when(resultSet.getLong("modified_by_user_id")).thenReturn(1L);
        when(resultSet.getTimestamp("modified_time")).thenReturn(Timestamp.from(Instant.now()));

        Address result = addressDaoPostgres.addressRowMapper(resultSet, 0);

        assertEquals("123 Main St", result.addressLine1());
        assertEquals("Apt 4B", result.addressLine2());
        assertEquals("Cityville", result.city());
        assertEquals("Stateville", result.stateProvince());
        assertEquals("12345", result.postalCode());
        assertEquals("Countryland", result.country());
    }
}