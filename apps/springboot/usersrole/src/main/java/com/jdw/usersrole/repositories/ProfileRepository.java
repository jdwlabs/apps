package com.jdw.usersrole.repositories;

import com.jdw.usersrole.models.Address;
import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.models.ProfileIcon;

import java.util.List;
import java.util.Optional;

public interface ProfileRepository {
    Optional<Profile> findById(Long id);

    Optional<Profile> findByUserId(Long id);

    List<Profile> findAll();

    Profile save(Profile profile);

    void deleteById(Long id);

    void deleteByUserId(Long id);

    Profile saveAddress(Address address);

    void deleteAddressByAddressId(Long addressId);

    Profile saveIcon(ProfileIcon icon);

    void deleteIconById(Long id);
}
