package com.jdw.usersrole.controllers;

import com.jdw.usersrole.models.SecurityUser;
import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.ProfileService;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.access.AccessDeniedException;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.junit.jupiter.SpringExtension;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.reset;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

// The @PreAuthorize expression is only enforced when the method-security interceptor
// proxies the controller, which standalone MockMvc does not set up. This boots just
// enough context for that interceptor to run.
@ExtendWith(SpringExtension.class)
@ContextConfiguration(classes = ProfileAddressAuthorizationTests.MethodSecurityTestConfig.class)
@Tag("fast")
@Tag("unit")
class ProfileAddressAuthorizationTests {
    private static final long TARGET_PROFILE_ID = 1L;
    private static final long TARGET_ADDRESS_ID = 10L;
    private static final long OTHER_PROFILE_ID = 2L;
    private static final String REQUESTER = "user@jdw.com";

    @Configuration
    @EnableMethodSecurity
    static class MethodSecurityTestConfig {
        @Bean
        ProfileService profileService() {
            return mock(ProfileService.class);
        }

        @Bean
        JwtService jwtService() {
            return mock(JwtService.class);
        }

        @Bean
        ProfilesController profilesController(ProfileService profileService, JwtService jwtService) {
            return new ProfilesController(profileService, jwtService);
        }
    }

    @Autowired
    private ProfilesController profilesController;
    @Autowired
    private ProfileService profileService;
    @Autowired
    private JwtService jwtService;

    @AfterEach
    void tearDown() {
        SecurityContextHolder.clearContext();
        reset(profileService, jwtService);
    }

    @BeforeEach
    void setUp() {
        SecurityContextHolder.clearContext();
    }

    private void authenticate(Long principalProfileId, String authority) {
        SecurityUser principal = mock(SecurityUser.class);
        when(principal.getProfileId()).thenReturn(principalProfileId);
        SecurityContextHolder.getContext().setAuthentication(new UsernamePasswordAuthenticationToken(
                principal, null, List.of(new SimpleGrantedAuthority(authority))));
    }

    @Test
    void deleteAddress_shouldBePermitted_forAnAdministratorTargetingAnotherProfile() {
        authenticate(OTHER_PROFILE_ID, "ADMIN");
        when(jwtService.getEmailAddress(anyString())).thenReturn(REQUESTER);

        profilesController.deleteAddress(TARGET_PROFILE_ID, TARGET_ADDRESS_ID, "Bearer token");

        verify(profileService).deleteAddress(TARGET_PROFILE_ID, TARGET_ADDRESS_ID, REQUESTER);
    }

    @Test
    void deleteAddress_shouldBePermitted_forTheOwnerOfTheProfileInThePath() {
        authenticate(TARGET_PROFILE_ID, "USER");
        when(jwtService.getEmailAddress(anyString())).thenReturn(REQUESTER);

        profilesController.deleteAddress(TARGET_PROFILE_ID, TARGET_ADDRESS_ID, "Bearer token");

        verify(profileService).deleteAddress(TARGET_PROFILE_ID, TARGET_ADDRESS_ID, REQUESTER);
    }

    @Test
    void deleteAddress_shouldBeDenied_forANonAdministratorWhoDoesNotOwnTheProfile() {
        authenticate(OTHER_PROFILE_ID, "USER");

        assertThrows(AccessDeniedException.class,
                () -> profilesController.deleteAddress(TARGET_PROFILE_ID, TARGET_ADDRESS_ID, "Bearer token"));

        verify(profileService, never()).deleteAddress(anyLong(), anyLong(), anyString());
    }
}
