package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Profile;

import java.util.List;
import java.util.Optional;

public interface ProfileDao {
    Profile create(Profile profile);

    Optional<Profile> findById(Long id);

    Optional<Profile> findByUserId(Long id);

    List<Profile> findAll();

    Profile update(Profile profile);

    void deleteById(Long id);

    void deleteByUserId(Long id);
}
