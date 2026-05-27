package com.jdw.usersrole.repositories;

import com.jdw.usersrole.daos.RoleDao;
import com.jdw.usersrole.daos.UserRoleDao;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.UserRole;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.Collections;
import java.util.HashSet;
import java.util.List;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class RoleRepositoryImplTests {
    @Mock
    private RoleDao roleDao;
    @Mock
    private UserRoleDao userRoleDao;
    @InjectMocks
    private RoleRepositoryImpl roleRepository;

    private Role buildMockRole() {
        return Role.builder()
                .id(1L)
                .name("Admin")
                .description("Administrator")
                .status("ACTIVE")
                .users(new HashSet<>())
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    private UserRole buildMockUserRole() {
        return UserRole.builder()
                .roleId(1L)
                .userId(1L)
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void findById_shouldReturnRoleWithUserRoles() {
        Role mockRole = buildMockRole();
        UserRole mockUserRole = buildMockUserRole();

        when(roleDao.findById(1L)).thenReturn(Optional.of(mockRole));
        when(userRoleDao.findByRoleId(1L)).thenReturn(List.of(mockUserRole));

        Optional<Role> result = roleRepository.findById(1L);

        assertTrue(result.isPresent());
        assertEquals(mockRole.id(), result.get().id());
        assertEquals(1, result.get().users().size());
        verify(roleDao, times(1)).findById(1L);
        verify(userRoleDao, times(1)).findByRoleId(1L);
    }

    @Test
    void findById_shouldEmptyWhenNotFound() {
        when(roleDao.findById(1L)).thenReturn(Optional.empty());

        Optional<Role> result = roleRepository.findById(1L);

        assertTrue(result.isEmpty());
    }

    @Test
    void findByName_shouldReturnRoleWithUserRoles() {
        Role mockRole = buildMockRole();
        UserRole mockUserRole = buildMockUserRole();

        when(roleDao.findByName("Admin")).thenReturn(Optional.of(mockRole));
        when(userRoleDao.findByRoleId(1L)).thenReturn(List.of(mockUserRole));

        Optional<Role> result = roleRepository.findByName("Admin");

        assertTrue(result.isPresent());
        assertEquals(mockRole.name(), result.get().name());
        assertEquals(1, result.get().users().size());
        verify(roleDao, times(1)).findByName("Admin");
        verify(userRoleDao, times(1)).findByRoleId(1L);
    }

    @Test
    void findAll_shouldReturnAllRolesWithUserRoles() {
        Role mockRole = buildMockRole();
        UserRole mockUserRole = buildMockUserRole();

        when(roleDao.findAll()).thenReturn(List.of(mockRole));
        when(userRoleDao.findByRoleId(1L)).thenReturn(List.of(mockUserRole));

        List<Role> result = roleRepository.findAll();

        assertEquals(1, result.size());
        assertEquals(mockRole.id(), result.getFirst().id());
        assertEquals(1, result.getFirst().users().size());
        verify(roleDao, times(1)).findAll();
        verify(userRoleDao, times(1)).findByRoleId(1L);
    }

    @Test
    void save_shouldCreateNewRoleWhenIdIsNull() {
        Role roleToSave = Role.builder()
                .name("Admin")
                .description("Administrator")
                .status("ACTIVE")
                .users(new HashSet<>())
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();

        when(roleDao.create(roleToSave)).thenReturn(roleToSave);

        Role result = roleRepository.save(roleToSave);

        assertEquals(roleToSave, result);
        verify(roleDao, times(1)).create(roleToSave);
        verify(roleDao, never()).update(any(Role.class));
    }

    @Test
    void save_shouldUpdateRoleWhenIdIsPresent() {
        Role mockRole = buildMockRole();

        when(roleDao.update(mockRole)).thenReturn(mockRole);
        when(userRoleDao.findByRoleId(1L)).thenReturn(Collections.emptyList());

        Role result = roleRepository.save(mockRole);

        assertEquals(mockRole, result);
        verify(roleDao, times(1)).update(mockRole);
        verify(roleDao, never()).create(any(Role.class));
    }

    @Test
    void save_shouldReturnNullWhenRoleIsNull() {
        Role result = roleRepository.save(null);

        assertNull(result);
        verify(roleDao, never()).create(any());
        verify(roleDao, never()).update(any());
    }

    @Test
    void deleteById_shouldDeleteRoleAndRelatedUserRoles() {
        roleRepository.deleteById(1L);

        verify(userRoleDao, times(1)).deleteByRoleId(1L);
        verify(roleDao, times(1)).deleteById(1L);
    }

    @Test
    void getRole_shouldReturnOptionalOfRoleWithUserRoles() {
        Role mockRole = buildMockRole();
        UserRole mockUserRole = buildMockUserRole();

        when(roleDao.findById(1L)).thenReturn(Optional.of(mockRole));
        when(userRoleDao.findByRoleId(1L)).thenReturn(List.of(mockUserRole));

        Optional<Role> result = roleRepository.findById(1L);

        assertTrue(result.isPresent());
        assertEquals(mockRole.id(), result.get().id());
        assertEquals(1, result.get().users().size());
        verify(roleDao, times(1)).findById(1L);
        verify(userRoleDao, times(1)).findByRoleId(1L);
    }

    @Test
    void grantUsers_ShouldAddUsersToRole() {
        Role mockRole = buildMockRole();
        UserRole userRoleToAdd = buildMockUserRole();
        List<UserRole> userRoleList = List.of(userRoleToAdd);

        when(userRoleDao.findByRoleIdAndUserId(anyLong(), anyLong())).thenReturn(Optional.empty());
        when(roleDao.findById(anyLong())).thenReturn(Optional.of(mockRole));
        when(userRoleDao.create(any(UserRole.class))).thenReturn(userRoleToAdd);

        Role result = roleRepository.grantUsers(userRoleList);

        assertNotNull(result);
        assertEquals(mockRole.id(), result.id());
        verify(userRoleDao, times(1)).findByRoleIdAndUserId(anyLong(), anyLong());
        verify(userRoleDao, times(1)).create(userRoleToAdd);
    }

    @Test
    void grantUsers_ShouldNotAddExistingUsers() {
        Role mockRole = buildMockRole();
        UserRole existingUserRole = buildMockUserRole();
        List<UserRole> userRoleList = List.of(existingUserRole);

        when(userRoleDao.findByRoleIdAndUserId(anyLong(), anyLong())).thenReturn(Optional.of(existingUserRole));
        when(roleDao.findById(anyLong())).thenReturn(Optional.of(mockRole));

        Role result = roleRepository.grantUsers(userRoleList);

        assertNotNull(result);
        assertEquals(mockRole.id(), result.id());
        verify(userRoleDao, times(1)).findByRoleIdAndUserId(anyLong(), anyLong());
        verify(userRoleDao, never()).create(any(UserRole.class));
    }

    @Test
    void revokeUsers_ShouldRemoveUsersFromRole() {
        Role mockRole = buildMockRole();
        UserRole userRoleToRemove = buildMockUserRole();
        List<UserRole> userRoleList = List.of(userRoleToRemove);

        when(userRoleDao.findByRoleIdAndUserId(anyLong(), anyLong())).thenReturn(Optional.of(userRoleToRemove));
        when(roleDao.findById(anyLong())).thenReturn(Optional.of(mockRole));
        doNothing().when(userRoleDao).deleteByRoleIdAndUserId(anyLong(), anyLong());

        Role result = roleRepository.revokeUsers(userRoleList);

        assertNotNull(result);
        assertEquals(mockRole.id(), result.id());
        verify(userRoleDao, times(1)).findByRoleIdAndUserId(anyLong(), anyLong());
        verify(userRoleDao, times(1)).deleteByRoleIdAndUserId(anyLong(), anyLong());
    }

    @Test
    void revokeUsers_ShouldNotRemoveNonExistingUsers() {
        Role mockRole = buildMockRole();
        UserRole nonExistingUserRole = buildMockUserRole();
        List<UserRole> userRoleList = List.of(nonExistingUserRole);

        when(userRoleDao.findByRoleIdAndUserId(anyLong(), anyLong())).thenReturn(Optional.empty());
        when(roleDao.findById(anyLong())).thenReturn(Optional.of(mockRole));

        Role result = roleRepository.revokeUsers(userRoleList);

        assertNotNull(result);
        assertEquals(mockRole.id(), result.id());
        verify(userRoleDao, times(1)).findByRoleIdAndUserId(anyLong(), anyLong());
        verify(userRoleDao, never()).deleteByRoleIdAndUserId(anyLong(), anyLong());
    }
}