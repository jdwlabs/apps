package com.jdw.usersrole.repositories;

import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.UserRole;

import java.util.List;
import java.util.Optional;

public interface RoleRepository {
    Optional<Role> findById(Long id);

    Optional<Role> findByName(String roleName);

    List<Role> findAll();

    Role save(Role role);

    void deleteById(Long id);

    Role grantUsers(List<UserRole> userRoleList);

    Role revokeUsers(List<UserRole> userRoleList);
}
