package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.RoleRequestDTO;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.services.RoleService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequiredArgsConstructor
@RequestMapping("/api/roles")
@Slf4j
public class RolesController {
    private final RoleService roleService;

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

    @PostMapping
    public ResponseEntity<Role> createRole(@Valid @RequestBody RoleRequestDTO role) {
        log.trace("Creating role {}", role);
        Role createdRole = roleService.createRole(role);
        return ResponseEntity.status(HttpStatus.CREATED).body(createdRole);
    }

    @PutMapping("/{roleId}")
    public ResponseEntity<Role> updateRole(@PathVariable Long roleId, @Valid @RequestBody RoleRequestDTO role) {
        log.trace("Updating role {}", role);
        Role updatedRole = roleService.updateRole(roleId, role);
        return ResponseEntity.ok(updatedRole);
    }

    @DeleteMapping("/{roleId}")
    public ResponseEntity<Void> deleteRole(@PathVariable Long roleId) {
        log.trace("Deleting role with id {}", roleId);
        roleService.deleteRole(roleId);
        return ResponseEntity.noContent().build();
    }
}
