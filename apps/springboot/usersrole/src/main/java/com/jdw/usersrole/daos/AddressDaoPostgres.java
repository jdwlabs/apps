package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Address;
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
public class AddressDaoPostgres implements AddressDao {
    private final JdbcClient jdbcClient;

    @Override
    public Address create(Address address) {
        log.debug("Creating address: {}", address);
        String sql = """
                INSERT INTO auth.addresses (profile_id, address_line_1, address_line_2, city, state_province, postal_code, country,
                                            created_by_user_id, created_time, modified_by_user_id, modified_time)
                VALUES (:profileId, :addressLine1, :addressLine2, :city, :stateProvince, :postalCode, :country, :createdByUserId,
                        :createdTime, :modifiedByUserId, :modifiedTime)
                RETURNING address_id
                """;
        Timestamp currentTime = Timestamp.from(Instant.now());
        Long generatedId = jdbcClient.sql(sql)
                .param("profileId", address.profileId())
                .param("addressLine1", address.addressLine1())
                .param("addressLine2", address.addressLine2())
                .param("city", address.city())
                .param("stateProvince", address.stateProvince())
                .param("postalCode", address.postalCode())
                .param("country", address.country())
                .param("createdByUserId", address.createdByUserId())
                .param("createdTime", currentTime)
                .param("modifiedByUserId", address.modifiedByUserId())
                .param("modifiedTime", currentTime)
                .query(Long.class)
                .single();
        return findById(generatedId).orElse(null);
    }

    @Override
    public Optional<Address> findById(Long id) {
        log.debug("Finding address with id: {}", id);
        String sql = "SELECT * FROM auth.addresses WHERE address_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(this::addressRowMapper)
                .optional();
    }

    @Override
    public List<Address> findByProfileId(Long id) {
        log.debug("Finding addresses with profile id: {}", id);
        String sql = "SELECT * FROM auth.addresses WHERE profile_id = :id";
        return jdbcClient.sql(sql)
                .param("id", id)
                .query(this::addressRowMapper)
                .list();
    }

    @Override
    public List<Address> findAll() {
        log.debug("Finding all addresses");
        String sql = "SELECT * FROM auth.addresses";
        return jdbcClient.sql(sql)
                .query(this::addressRowMapper)
                .list();
    }

    @Override
    public Address update(Address address) {
        log.debug("Updating address: {}", address);
        String sql = """
                UPDATE auth.addresses
                SET address_line_1      = :addressLine1,
                    address_line_2      = :addressLine2,
                    city                = :city,
                    state_province      = :stateProvince,
                    postal_code         = :postalCode,
                    country             = :country,
                    modified_by_user_id = :modifiedByUserId,
                    modified_time       = :modifiedTime
                WHERE address_id = :id
                """;
        jdbcClient.sql(sql)
                .param("addressLine1", address.addressLine1())
                .param("addressLine2", address.addressLine2())
                .param("city", address.city())
                .param("stateProvince", address.stateProvince())
                .param("postalCode", address.postalCode())
                .param("country", address.country())
                .param("modifiedByUserId", address.modifiedByUserId())
                .param("modifiedTime", address.modifiedTime())
                .param("id", address.id())
                .update();
        return findById(address.id()).orElse(null);
    }

    @Override
    public void deleteById(Long id) {
        log.debug("Deleting address with id: {}", id);
        String sql = "DELETE FROM auth.addresses WHERE address_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    @Override
    public void deleteByProfileId(Long id) {
        log.debug("Deleting addresses with profile id: {}", id);
        String sql = "DELETE FROM auth.addresses WHERE profile_id = :id";
        jdbcClient.sql(sql)
                .param("id", id)
                .update();
    }

    Address addressRowMapper(ResultSet rs, int rowNum) throws SQLException {
        log.debug("Mapping address: rs={}, rowNum={}", rs, rowNum);
        Long id = rs.getLong("address_id");
        Long profileId = rs.getLong("profile_id");
        String addressLine1 = rs.getString("address_line_1");
        String addressLine2 = rs.getString("address_line_2");
        String city = rs.getString("city");
        String stateProvince = rs.getString("state_province");
        String postalCode = rs.getString("postal_code");
        String country = rs.getString("country");
        Long createdByUserId = rs.getLong("created_by_user_id");
        Timestamp createdTime = rs.getTimestamp("created_time");
        Long modifiedByUserId = rs.getLong("modified_by_user_id");
        Timestamp modifiedTime = rs.getTimestamp("modified_time");
        return Address.builder()
                .id(id)
                .addressLine1(addressLine1)
                .addressLine2(addressLine2)
                .city(city)
                .stateProvince(stateProvince)
                .postalCode(postalCode)
                .country(country)
                .profileId(profileId)
                .createdByUserId(createdByUserId)
                .createdTime(createdTime)
                .modifiedByUserId(modifiedByUserId)
                .modifiedTime(modifiedTime)
                .build();
    }
}
