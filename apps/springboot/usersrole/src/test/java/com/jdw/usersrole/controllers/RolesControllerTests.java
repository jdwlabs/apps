package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.RoleRequestDTO;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.UserRole;
import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.RoleService;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;

import java.sql.Timestamp;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class RolesControllerTests {
    @Mock
    private RoleService roleService;
    @Mock
    private JwtService jwtService;
    @InjectMocks
    private RolesController rolesController;

    private Role buildMockRole() {
        Set<UserRole> mockUsers = new HashSet<>();
        return Role.builder()
                .id(1L)
                .name("ROLE_ADMIN")
                .description("Administrator role")
                .status("ACTIVE")
                .users(mockUsers)
                .createdByUserId(1L)
                .createdTime(new Timestamp(System.currentTimeMillis()))
                .modifiedByUserId(1L)
                .modifiedTime(new Timestamp(System.currentTimeMillis()))
                .build();
    }

    @Test
    void getAllRoles_shouldReturnListOfRoles() {
        Role mockRole = buildMockRole();
        when(roleService.getAllRoles()).thenReturn(List.of(mockRole));

        ResponseEntity<List<Role>> response = rolesController.getAllRoles();

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertNotNull(response.getBody());
        assertEquals(1, response.getBody().size());
        assertEquals(mockRole, response.getBody().getFirst());
    }

    @Test
    void getRoleById_shouldReturnRole() {
        Long roleId = 1L;
        Role mockRole = buildMockRole();
        when(roleService.getRoleById(roleId)).thenReturn(mockRole);

        ResponseEntity<Role> response = rolesController.getRoleById(roleId);

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(mockRole, response.getBody());
    }

    @Test
    void getRoleByName_shouldReturnRole() {
        String roleName = "ROLE_ADMIN";
        Role mockRole = buildMockRole();
        when(roleService.getRoleByName(roleName)).thenReturn(mockRole);

        ResponseEntity<Role> response = rolesController.getRoleByName(roleName);

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(mockRole, response.getBody());
    }

    @Test
    void createRole_shouldReturnCreatedRole() {
        RoleRequestDTO request = new RoleRequestDTO("ROLE_ADMIN", "Administrator role");
        Role createdRole = buildMockRole();
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(roleService.createRole(any(RoleRequestDTO.class), any(String.class))).thenReturn(createdRole);

        ResponseEntity<Role> response = rolesController.createRole(request, "Bearer token");

        assertEquals(HttpStatus.CREATED, response.getStatusCode());
        assertEquals(createdRole, response.getBody());
    }

    @Test
    void updateRole_shouldReturnUpdatedRole() {
        Long roleId = 1L;
        RoleRequestDTO request = new RoleRequestDTO("ROLE_ADMIN", "Updated administrator role");
        Role updatedRole = buildMockRole();
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("admin@jdw.com");
        when(roleService.updateRole(any(Long.class), any(RoleRequestDTO.class), any(String.class))).thenReturn(updatedRole);

        ResponseEntity<Role> response = rolesController.updateRole(roleId, request, "Bearer token");

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(updatedRole, response.getBody());
    }

    @Test
    void deleteRole_shouldReturnNoContent() {
        Long roleId = 1L;
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");

        ResponseEntity<Void> response = rolesController.deleteRole(roleId, "Bearer token");

        assertEquals(HttpStatus.NO_CONTENT, response.getStatusCode());
        verify(roleService).deleteRole(roleId, "user@jdw.com");
    }
}