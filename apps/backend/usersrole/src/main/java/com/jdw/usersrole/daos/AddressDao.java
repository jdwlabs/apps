package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Address;

import java.util.List;
import java.util.Optional;

public interface AddressDao {
    Address create(Address address);

    Optional<Address> findById(Long id);

    List<Address> findByProfileId(Long id);

    List<Address> findAll();

    Address update(Address address);

    int deleteByIdAndProfileId(Long addressId, Long profileId);

    void deleteByProfileId(Long id);
}
