package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.RoleRequestDTO;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.RoleService;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotEmpty;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequiredArgsConstructor
@RequestMapping("/api/roles")
@Slf4j
public class RolesController {
    private final RoleService roleService;
    private final JwtService jwtService;

    @GetMapping
    public ResponseEntity<List<Role>> getAllRoles() {
        log.trace("Getting all roles");
        return ResponseEntity.ok(roleService.getAllRoles());
    }

    @GetMapping("/{roleId}")
    public ResponseEntity<Role> getRoleById(@PathVariable Long roleId) {
        log.trace("Getting role with id {}", roleId);
        Role role = roleService.getRoleById(roleId);
        return ResponseEntity.ok(role);
    }

    @GetMapping("/name/{roleName}")
    public ResponseEntity<Role> getRoleByName(@PathVariable String roleName) {
        log.trace("Getting role with name {}", roleName);
        Role role = roleService.getRoleByName(roleName);
        return ResponseEntity.ok(role);
    }

    @PreAuthorize("hasAuthority('ADMIN')")
    @PostMapping
    public ResponseEntity<Role> createRole(@Valid @RequestBody RoleRequestDTO role, @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Creating role {}", role);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        Role createdRole = roleService.createRole(role, emailAddress);
        return ResponseEntity.status(HttpStatus.CREATED).body(createdRole);
    }

    @PreAuthorize("hasAuthority('ADMIN')")
    @PutMapping("/{roleId}")
    public ResponseEntity<Role> updateRole(@PathVariable Long roleId, @Valid @RequestBody RoleRequestDTO role, @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Updating role {}", role);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        Role updatedRole = roleService.updateRole(roleId, role, emailAddress);
        return ResponseEntity.ok(updatedRole);
    }

    @PreAuthorize("hasAuthority('ADMIN')")
    @DeleteMapping("/{roleId}")
    public ResponseEntity<Void> deleteRole(@PathVariable Long roleId, @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Deleting role with id {}", roleId);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        roleService.deleteRole(roleId, emailAddress);
        return ResponseEntity.noContent().build();
    }

    @PreAuthorize("hasAnyAuthority('ADMIN', 'MANAGER')")
    @PutMapping("/{roleId}/users/grant")
    public ResponseEntity<Role> grantUsersToRole(
            @PathVariable Long roleId,
            @NotEmpty @RequestBody List<Long> userIds,
            @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Granting users {} to role with id {}", userIds, roleId);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        Role updatedRole = roleService.grantUsersToRole(roleId, userIds, emailAddress);
        return ResponseEntity.ok(updatedRole);
    }

    @PreAuthorize("hasAnyAuthority('ADMIN', 'MANAGER')")
    @PutMapping("/{roleId}/users/revoke")
    public ResponseEntity<Role> revokeUsersFromRole(
            @PathVariable Long roleId,
            @NotEmpty @RequestBody List<Long> userIds,
            @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Revoking users {} from role with id {}", userIds, roleId);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        Role updatedRole = roleService.revokeUsersFromRole(roleId, userIds, emailAddress);
        return ResponseEntity.ok(updatedRole);
    }
}
