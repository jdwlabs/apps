package com.jdw.usersrole.controllers;

import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.services.RoleService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequiredArgsConstructor
@RequestMapping("/api/roles")
@Slf4j
public class RolesController {
    private final RoleService roleService;

    @GetMapping
    public List<Role> getAllRoles() {
        log.trace("Getting all roles");
        return roleService.getAllRoles();
    }

    @GetMapping("/{roleId}")
    public Role getRoleById(@PathVariable Long roleId) {
        log.trace("Getting role by id {}", roleId);
        return roleService.getRoleById(roleId);
    }
}
