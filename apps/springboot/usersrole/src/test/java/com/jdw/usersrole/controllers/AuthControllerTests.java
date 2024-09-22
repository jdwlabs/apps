package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.AuthResponseDTO;
import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.services.AuthService;
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

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class AuthControllerTests {
    @Mock
    private AuthService authService;
    @Mock
    private UserService userService;
    @InjectMocks
    private AuthController authController;

    @Test
    void authenticate_shouldReturnToken() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", "P@ssw0rd!");
        AuthResponseDTO mockResponse = AuthResponseDTO.builder().jwtToken("mock-jwt-token").build();
        when(authService.authenticate(userRequestDTO)).thenReturn(mockResponse);

        ResponseEntity<AuthResponseDTO> response = authController.authenticate(userRequestDTO);

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(mockResponse, response.getBody());
    }

    @Test
    void createUser_shouldReturnCreatedUser() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", "P@ssw0rd!");
        User mockUser = buildMockUser();
        when(userService.createUser(any(UserRequestDTO.class))).thenReturn(mockUser);

        ResponseEntity<User> response = authController.createUser(userRequestDTO);

        assertEquals(HttpStatus.CREATED, response.getStatusCode());
        assertEquals(mockUser, response.getBody());
    }

    private User buildMockUser() {
        return User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .password("encrypted-password")
                .status("ACTIVE")
                .roles(null)
                .profile(null)
                .createdByUserId(1L)
                .createdTime(new Timestamp(System.currentTimeMillis()))
                .modifiedByUserId(1L)
                .modifiedTime(new Timestamp(System.currentTimeMillis()))
                .build();
    }

}