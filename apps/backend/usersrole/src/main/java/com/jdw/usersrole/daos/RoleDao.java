package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.Role;

import java.util.List;
import java.util.Optional;

public interface RoleDao {
    Role create(Role role);

    Optional<Role> findById(Long id);

    Optional<Role> findByName(String name);

    List<Role> findAll();

    Role update(Role role);

    void deleteById(Long id);
}
