package com.jdw.usersrole.models;

import com.jdw.usersrole.repositories.RoleRepository;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;

import java.util.Collection;
import java.util.Optional;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class SecurityUserTests {
    @Mock
    private RoleRepository roleRepository;
    @Mock
    private User user;
    @InjectMocks
    private SecurityUser securityUser;

    @Test
    void getAuthorities_shouldReturnAuthoritiesFromRoles() {
        Role role1 = Role.builder()
                .id(1L)
                .name("ROLE_USER")
                .description("User role")
                .status("ACTIVE")
                .users(Set.of())
                .createdByUserId(null)
                .createdTime(null)
                .modifiedByUserId(null)
                .modifiedTime(null)
                .build();

        Role role2 = Role.builder()
                .id(2L)
                .name("ROLE_ADMIN")
                .description("Admin role")
                .status("ACTIVE")
                .users(Set.of())
                .createdByUserId(null)
                .createdTime(null)
                .modifiedByUserId(null)
                .modifiedTime(null)
                .build();

        UserRole userRole1 = UserRole.builder()
                .roleId(1L)
                .userId(1L)
                .build();

        UserRole userRole2 = UserRole.builder()
                .roleId(2L)
                .userId(2L)
                .build();

        when(user.roles()).thenReturn(Set.of(userRole1, userRole2));
        when(roleRepository.findById(1L)).thenReturn(Optional.of(role1));
        when(roleRepository.findById(2L)).thenReturn(Optional.of(role2));

        Collection<? extends GrantedAuthority> authorities = securityUser.getAuthorities();

        assertEquals(2, authorities.size());
        assertTrue(authorities.contains(new SimpleGrantedAuthority("ROLE_USER")));
        assertTrue(authorities.contains(new SimpleGrantedAuthority("ROLE_ADMIN")));
    }

    @Test
    void getPassword_shouldReturnUserPassword() {
        String expectedPassword = "password";
        when(user.password()).thenReturn(expectedPassword);

        String actualPassword = securityUser.getPassword();

        assertEquals(expectedPassword, actualPassword);
    }

    @Test
    void getUsername_shouldReturnUserEmailAddress() {
        String expectedEmail = "user@example.com";
        when(user.emailAddress()).thenReturn(expectedEmail);

        String actualEmail = securityUser.getUsername();

        assertEquals(expectedEmail, actualEmail);
    }

    @Test
    void isAccountNonExpired_shouldReturnTrue() {
        boolean isNonExpired = securityUser.isAccountNonExpired();

        assertTrue(isNonExpired);
    }

    @Test
    void isAccountNonLocked_shouldReturnTrue() {
        boolean isNonLocked = securityUser.isAccountNonLocked();

        assertTrue(isNonLocked);
    }

    @Test
    void isCredentialsNonExpired_shouldReturnTrue() {
        boolean areCredentialsNonExpired = securityUser.isCredentialsNonExpired();

        assertTrue(areCredentialsNonExpired);
    }

    @Test
    void isEnabled_shouldReturnTrue() {
        boolean isEnabled = securityUser.isEnabled();

        assertTrue(isEnabled);
    }

    @Test
    void getUserId_shouldReturnUserId() {
        Long expectedUserId = 123L;
        when(user.id()).thenReturn(expectedUserId);

        Long actualUserId = securityUser.getUserId();

        assertEquals(expectedUserId, actualUserId);
    }

    @Test
    void getProfileId_shouldReturnProfileId() {
        Long expectedProfileId = 456L;
        Profile profile = mock(Profile.class);
        when(profile.id()).thenReturn(expectedProfileId);
        when(user.profile()).thenReturn(profile);

        Long actualProfileId = securityUser.getProfileId();

        assertEquals(expectedProfileId, actualProfileId);
    }
}