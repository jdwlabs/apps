package com.jdw.usersrole.services;

import com.jdw.usersrole.dtos.RoleRequestDTO;
import com.jdw.usersrole.exceptions.ResourceExistsException;
import com.jdw.usersrole.exceptions.ResourceNotFoundException;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.repositories.RoleRepository;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.List;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class RoleServiceTests {
    @Mock
    private RoleRepository roleRepository;
    @Mock
    private UserService userService;
    @InjectMocks
    private RoleService roleService;

    @Test
    void getAllRoles_ShouldReturnListOfRoles() {
        Role role = Role.builder()
                .id(1L)
                .name("ADMIN")
                .description("Administrator")
                .status(Status.ACTIVE.name())
                .build();
        when(roleRepository.findAll()).thenReturn(List.of(role));

        List<Role> roles = roleService.getAllRoles();

        assertNotNull(roles);
        assertEquals(1, roles.size());
        assertEquals("ADMIN", roles.getFirst().name());
    }

    @Test
    void getRoleById_ShouldReturnRole_WhenRoleExists() {
        Role role = Role.builder()
                .id(1L)
                .name("USER")
                .description("Standard User")
                .status(Status.ACTIVE.name())
                .build();
        when(roleRepository.findById(1L)).thenReturn(Optional.of(role));

        Role foundRole = roleService.getRoleById(1L);

        assertNotNull(foundRole);
        assertEquals(1L, foundRole.id());
        assertEquals("USER", foundRole.name());
    }

    @Test
    void getRoleById_ShouldThrowException_WhenRoleNotFound() {
        when(roleRepository.findById(anyLong())).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> roleService.getRoleById(1L));

        assertEquals("Role not found with id 1", exception.getMessage());
    }

    @Test
    void getRoleByName_ShouldReturnRole_WhenRoleExists() {
        Role role = Role.builder()
                .id(1L)
                .name("USER")
                .description("Standard User")
                .status(Status.ACTIVE.name())
                .build();
        when(roleRepository.findByName("USER")).thenReturn(Optional.of(role));

        Role foundRole = roleService.getRoleByName("USER");

        assertNotNull(foundRole);
        assertEquals(1L, foundRole.id());
        assertEquals("USER", foundRole.name());
    }

    @Test
    void getRoleByName_ShouldThrowException_WhenRoleNotFound() {
        when(roleRepository.findByName(anyString())).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> roleService.getRoleByName("ADMIN"));

        assertEquals("Role not found with name ADMIN", exception.getMessage());
    }

    @Test
    void createRole_ShouldCreateAndReturnRole() {
        RoleRequestDTO roleRequestDTO = new RoleRequestDTO("MANAGER", "Manages things");
        when(roleRepository.findByName(anyString())).thenReturn(Optional.empty());
        when(userService.getUserIdByEmailAddress(anyString())).thenReturn(1L);
        when(roleRepository.save(any(Role.class))).thenAnswer(invocation -> invocation.getArgument(0));

        Role createdRole = roleService.createRole(roleRequestDTO, "user@example.com");

        assertNotNull(createdRole);
        assertEquals("MANAGER", createdRole.name());
        assertEquals("Manages things", createdRole.description());
        assertEquals(Status.ACTIVE.name(), createdRole.status());
        verify(roleRepository, times(1)).save(any(Role.class));
    }

    @Test
    void createRole_ShouldThrowException_WhenRoleAlreadyExists() {
        RoleRequestDTO roleRequestDTO = new RoleRequestDTO("MANAGER", "Manages things");
        when(roleRepository.findByName(anyString())).thenReturn(Optional.of(Role.builder().build()));

        ResourceExistsException exception = assertThrows(ResourceExistsException.class,
                () -> roleService.createRole(roleRequestDTO, "user@example.com"));

        assertEquals("Role already exists with name MANAGER", exception.getMessage());
    }

    @Test
    void updateRole_ShouldUpdateAndReturnRole() {
        Role existingRole = Role.builder()
                .id(1L)
                .name("MANAGER")
                .description("Old Description")
                .status(Status.ACTIVE.name())
                .build();
        RoleRequestDTO roleRequestDTO = new RoleRequestDTO("MANAGER", "New Description");

        when(roleRepository.findById(1L)).thenReturn(Optional.of(existingRole));
        when(userService.getUserIdByEmailAddress(anyString())).thenReturn(1L);
        when(roleRepository.save(any(Role.class))).thenAnswer(invocation -> invocation.getArgument(0));

        Role updatedRole = roleService.updateRole(1L, roleRequestDTO, "user@example.com");

        assertNotNull(updatedRole);
        assertEquals("MANAGER", updatedRole.name());
        assertEquals("New Description", updatedRole.description());
        assertEquals(existingRole.status(), updatedRole.status());
        verify(roleRepository, times(1)).save(any(Role.class));
    }

    @Test
    void updateRole_ShouldThrowException_WhenRoleNotFound() {
        RoleRequestDTO roleRequestDTO = new RoleRequestDTO("MANAGER", "New Description");
        when(roleRepository.findById(anyLong())).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> roleService.updateRole(1L, roleRequestDTO, "user@example.com"));

        assertEquals("Role not found with id 1", exception.getMessage());
    }

    @Test
    void deleteRole_ShouldDeleteRole_WhenRoleExists() {
        doNothing().when(roleRepository).deleteById(anyLong());

        roleService.deleteRole(1L, "user@example.com");

        verify(roleRepository, times(1)).deleteById(1L);
    }
}