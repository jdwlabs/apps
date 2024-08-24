package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.models.UserRole;
import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.UserService;
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
class UsersControllerTests {
    @Mock
    private UserService userService;
    @Mock
    private JwtService jwtService;
    @InjectMocks
    private UsersController usersController;

    private User buildMockUser() {
        Set<UserRole> mockRoles = new HashSet<>();

        return User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .password("P@ssw0rd!")
                .status("ACTIVE")
                .roles(mockRoles)
                .profile(null)
                .createdByUserId(1L)
                .createdTime(new Timestamp(System.currentTimeMillis()))
                .modifiedByUserId(1L)
                .modifiedTime(new Timestamp(System.currentTimeMillis()))
                .build();
    }

    @Test
    void getAllUsers_shouldReturnListOfUsers() {
        User mockUser = buildMockUser();
        when(userService.getAllUsers()).thenReturn(List.of(mockUser));

        ResponseEntity<List<User>> response = usersController.getAllUsers();

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertNotNull(response.getBody());
        assertEquals(1, response.getBody().size());
        assertEquals(mockUser, response.getBody().getFirst());
    }

    @Test
    void getUserById_shouldReturnUser() {
        Long userId = 1L;
        User mockUser = buildMockUser();
        when(userService.getUserById(userId)).thenReturn(mockUser);

        ResponseEntity<User> response = usersController.getUserById(userId);

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(mockUser, response.getBody());
    }

    @Test
    void getUserByEmailAddress_shouldReturnUser() {
        String emailAddress = "user@jdw.com";
        User mockUser = buildMockUser();
        when(userService.getUserByEmailAddress(emailAddress)).thenReturn(mockUser);

        ResponseEntity<User> response = usersController.getUserByEmailAddress(emailAddress);

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(mockUser, response.getBody());
    }

    @Test
    void createUser_shouldReturnCreatedUser() {
        UserRequestDTO userRequest = new UserRequestDTO("user@jdw.com", "P@ssw0rd!");
        User createdUser = buildMockUser();
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(userService.createUser(any(UserRequestDTO.class), any(String.class))).thenReturn(createdUser);

        ResponseEntity<User> response = usersController.createUser(userRequest, "Bearer token");

        assertEquals(HttpStatus.CREATED, response.getStatusCode());
        assertEquals(createdUser, response.getBody());
    }

    @Test
    void updateUser_shouldReturnUpdatedUser() {
        Long userId = 1L;
        UserRequestDTO userRequest = new UserRequestDTO("user@jdw.com", "password");
        User updatedUser = buildMockUser();
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("admin@example.com");
        when(userService.updateUser(any(Long.class), any(UserRequestDTO.class), any(String.class))).thenReturn(updatedUser);

        ResponseEntity<User> response = usersController.updateUser(userId, userRequest, "Bearer token");

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(updatedUser, response.getBody());
    }

    @Test
    void deleteUser_shouldReturnNoContent() {
        Long userId = 1L;
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");

        ResponseEntity<User> response = usersController.deleteUser(userId, "Bearer token");

        assertEquals(HttpStatus.NO_CONTENT, response.getStatusCode());
        verify(userService).deleteUser(userId, "user@jdw.com");
    }
}