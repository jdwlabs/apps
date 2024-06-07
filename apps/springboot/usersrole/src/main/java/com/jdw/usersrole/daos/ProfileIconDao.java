package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.ProfileIcon;

import java.util.List;
import java.util.Optional;

public interface ProfileIconDao {
    ProfileIcon create(ProfileIcon profileIcon);

    Optional<ProfileIcon> findById(Long id);

    Optional<ProfileIcon> findByProfileId(Long id);

    List<ProfileIcon> findAll();

    ProfileIcon update(ProfileIcon profileIcon);

    void deleteById(Long id);

    void deleteByProfileId(Long id);
}
