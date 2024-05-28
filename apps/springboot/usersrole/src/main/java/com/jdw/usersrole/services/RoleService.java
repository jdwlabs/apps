package com.jdw.usersrole.services;

import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.repositories.RoleRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class RoleService {
    private final RoleRepository roleRepository;

    public List<Role> getAllRoles() {
        log.info("Getting all roles");
        return roleRepository.findAll();
    }

    public Role getRoleById(Long id) {
        log.info("Getting role with id {}", id);
        return roleRepository.findById(id).orElse(null);
    }

    public Role getRoleByName(String name) {
        log.info("Getting role with name {}", name);
        return roleRepository.findByName(name).orElse(null);
    }

    public Role createRole(Role role) {
        log.info("Creating role {}", role);
        Timestamp currentTime = Timestamp.from(Instant.now());
        Role newRole = Role.builder()
                .name(role.name())
                .description(role.description())
                .status(Status.ACTIVE.name())
                .users(null)
                .createdByUserId(1L)
                .createdTime(currentTime)
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return roleRepository.save(newRole);
    }

    public Role updateRole(Long id, Role role) {
        log.info("Updating role: id={}, role={}", id, role);
        if (role == null
                || id == null
                || role.name() == null
                || role.description() == null
                || !roleRepository.existsById(id)) {
            return null;
        }
        Role currentRole = roleRepository.findById(id).orElse(null);
        if (currentRole == null) {
            return null;
        }
        String status = role.status() != null ? role.status() : Status.ACTIVE.name();
        Timestamp currentTime = Timestamp.from(Instant.now());
        Role updatedRole = Role.builder()
                .id(id)
                .name(role.name())
                .description(role.description())
                .status(status)
                .users(currentRole.users())
                .createdByUserId(currentRole.createdByUserId())
                .createdTime(currentRole.createdTime())
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return roleRepository.save(updatedRole);
    }

    @Transactional
    public void deleteRole(Long id) {
        log.info("Deleting role with id {}", id);
        roleRepository.deleteById(id);
    }
}
