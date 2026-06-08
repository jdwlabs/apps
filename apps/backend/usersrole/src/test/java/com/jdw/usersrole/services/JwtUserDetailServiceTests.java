package com.jdw.usersrole.services;

import com.jdw.usersrole.models.SecurityUser;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.repositories.UserRepository;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UsernameNotFoundException;

import java.sql.Timestamp;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class JwtUserDetailServiceTests {
    @Mock
    private UserRepository userRepository;
    @InjectMocks
    private JwtUserDetailService jwtUserDetailService;

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

    @Test
    void loadUserByUsername_shouldReturnUserDetails() {
        User user = buildMockUser();
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));

        UserDetails userDetails = jwtUserDetailService.loadUserByUsername("user@jdw.com");

        assertNotNull(userDetails, "UserDetails should not be null");
        assertInstanceOf(SecurityUser.class, userDetails, "UserDetails should be an instance of SecurityUser");
        assertEquals("user@jdw.com", userDetails.getUsername(), "Username should match the email address");
    }

    @Test
    void loadUserByUsername_shouldThrowUsernameNotFoundException() {
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.empty());

        UsernameNotFoundException thrownException = assertThrows(
                UsernameNotFoundException.class,
                () -> jwtUserDetailService.loadUserByUsername("user@jdw.com"),
                "Expected loadUserByUsername to throw UsernameNotFoundException"
        );

        assertEquals("Email address not found: emailAddress=user@jdw.com", thrownException.getMessage(), "Exception message should match");
    }

}