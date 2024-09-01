package com.jdw.usersrole.services;

import com.jdw.usersrole.dtos.RoleRequestDTO;
import com.jdw.usersrole.exceptions.ResourceExistsException;
import com.jdw.usersrole.exceptions.ResourceNotFoundException;
import com.jdw.usersrole.metrics.ExecutionTimeLogger;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.repositories.RoleRepository;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotNull;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class RoleService {
    private final RoleRepository roleRepository;
    private final UserService userService;

    public List<Role> getAllRoles() {
        log.info("Getting all roles");
        return roleRepository.findAll();
    }

    public Role getRoleById(@NotNull Long id) {
        log.info("Getting role with id {}", id);
        return roleRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Role not found with id " + id));
    }

    public Role getRoleByName(@NotNull String name) {
        log.info("Getting role with name {}", name);
        return roleRepository.findByName(name)
                .orElseThrow(() -> new ResourceNotFoundException("Role not found with name " + name));
    }

    @ExecutionTimeLogger
    public Role createRole(@NotNull @Valid RoleRequestDTO roleDTO, @NotNull String emailAddress) {
        log.info("Creating role {}", roleDTO);
        roleRepository.findByName(roleDTO.name()).ifPresent(role -> {
            throw new ResourceExistsException("Role already exists with name " + roleDTO.name());
        });
        Long userId = userService.getUserIdByEmailAddress(emailAddress);
        Timestamp currentTime = Timestamp.from(Instant.now());
        Role newRole = Role.builder()
                .id(null)
                .name(roleDTO.name())
                .description(roleDTO.description())
                .status(Status.ACTIVE.name())
                .users(null)
                .createdByUserId(userId)
                .createdTime(currentTime)
                .modifiedByUserId(userId)
                .modifiedTime(currentTime)
                .build();
        return roleRepository.save(newRole);
    }

    @ExecutionTimeLogger
    public Role updateRole(@NotNull Long id, @NotNull @Valid RoleRequestDTO roleDTO, @NotNull String emailAddress) {
        log.info("Updating role: id={}, role={}", id, roleDTO);
        Role currentRole = roleRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Role not found with id " + id));
        Long userId = userService.getUserIdByEmailAddress(emailAddress);
        Timestamp currentTime = Timestamp.from(Instant.now());
        Role updatedRole = Role.builder()
                .id(id)
                .name(roleDTO.name())
                .description(roleDTO.description())
                .status(currentRole.status())
                .users(currentRole.users())
                .createdByUserId(currentRole.createdByUserId())
                .createdTime(currentRole.createdTime())
                .modifiedByUserId(userId)
                .modifiedTime(currentTime)
                .build();
        return roleRepository.save(updatedRole);
    }

    @ExecutionTimeLogger
    public void deleteRole(@NotNull Long id, @NotNull String emailAddress) {
        log.info("Deleting role with: id={}, requester={}", id, emailAddress);
        roleRepository.deleteById(id);
    }
}
