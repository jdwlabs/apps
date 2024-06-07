package com.jdw.usersrole.repositories;

import com.jdw.usersrole.daos.RoleDao;
import com.jdw.usersrole.daos.UserRoleDao;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.UserRole;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

import java.util.HashSet;
import java.util.List;
import java.util.Optional;
import java.util.Set;

@Repository
@RequiredArgsConstructor
@Slf4j
public class RoleRepositoryImpl implements RoleRepository {
    private final RoleDao roleDao;
    private final UserRoleDao userRoleDao;

    @Override
    @Transactional(readOnly = true)
    public Optional<Role> findById(Long id) {
        log.debug("Finding role by id: {}", id);
        Optional<Role> role = roleDao.findById(id);
        return getRole(role);
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<Role> findByName(String name) {
        log.debug("Finding role by name: {}", name);
        Optional<Role> role = roleDao.findByName(name);
        return getRole(role);
    }

    @Override
    @Transactional(readOnly = true)
    public List<Role> findAll() {
        log.debug("Finding all roles");
        List<Role> roles = roleDao.findAll();
        return roles.stream().map(this::getRole).toList();
    }

    @Override
    @Transactional
    public Role save(Role role) {
        log.debug("Saving role: {}", role);
        Role updatedRole;
        if (role == null) {
            return null;
        } else if (role.id() == null) {
            log.debug("Saving new role: {}", role);
            updatedRole = roleDao.create(role);
        } else {
            log.debug("Updating role: {}", role);
            updatedRole = roleDao.update(role);
        }
        return getRole(updatedRole);
    }

    @Override
    @Transactional
    public void deleteById(Long id) {
        log.debug("Deleting role with id {}", id);
        userRoleDao.deleteByRoleId(id);
        roleDao.deleteById(id);
    }

    @Override
    @Transactional(readOnly = true)
    public boolean existsById(Long id) {
        log.debug("Checking if role with id {} exists", id);
        return roleDao.findById(id).isPresent();
    }

    private Optional<Role> getRole(Optional<Role> role) {
        log.debug("Getting optional role: {}", role);
        if (role.isPresent()) {
            Role existingRole = role.get();
            List<UserRole> userRoles = userRoleDao.findByRoleId(existingRole.id());
            Set<UserRole> userRoleSet = new HashSet<>(userRoles);
            Role transformedRole = Role.builder()
                    .id(existingRole.id())
                    .name(existingRole.name())
                    .description(existingRole.description())
                    .status(existingRole.status())
                    .users(userRoleSet)
                    .createdByUserId(existingRole.createdByUserId())
                    .createdTime(existingRole.createdTime())
                    .modifiedByUserId(existingRole.modifiedByUserId())
                    .modifiedTime(existingRole.modifiedTime())
                    .build();
            return Optional.ofNullable(transformedRole);
        }
        return role;
    }

    private Role getRole(Role role) {
        log.debug("Getting role: {}", role);
        List<UserRole> userRoles = userRoleDao.findByRoleId(role.id());
        Set<UserRole> userRoleSet = new HashSet<>(userRoles);
        return Role.builder()
                .id(role.id())
                .name(role.name())
                .description(role.description())
                .status(role.status())
                .users(userRoleSet)
                .createdByUserId(role.createdByUserId())
                .createdTime(role.createdTime())
                .modifiedByUserId(role.modifiedByUserId())
                .modifiedTime(role.modifiedTime())
                .build();
    }
}
