package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.UserRole;

import java.util.List;

public interface UserRoleDao {
    UserRole create(UserRole userRole);

    List<UserRole> findByRoleId(Long roleId);

    List<UserRole> findByUserId(Long userId);

    void deleteByRoleId(Long roleId);

    void deleteByUserId(Long userId);
}
