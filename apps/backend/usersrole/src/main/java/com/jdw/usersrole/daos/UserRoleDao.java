package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.UserRole;

import java.util.List;
import java.util.Optional;

public interface UserRoleDao {
    UserRole create(UserRole userRole);

    List<UserRole> findByRoleId(Long roleId);

    List<UserRole> findByUserId(Long userId);

    Optional<UserRole> findByRoleIdAndUserId(Long roleId, Long userId);

    void deleteByRoleId(Long roleId);

    void deleteByUserId(Long userId);

    void deleteByRoleIdAndUserId(Long roleId, Long userId);
}
