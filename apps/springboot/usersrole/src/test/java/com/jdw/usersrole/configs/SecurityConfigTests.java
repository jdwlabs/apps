package com.jdw.usersrole.configs;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.CorsConfigurationSource;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class SecurityConfigTests {
    @InjectMocks
    private SecurityConfig securityConfig;

    @Test
    void testPasswordEncoderBean() {
        var passwordEncoder = securityConfig.passwordEncoder();
        assertNotNull(passwordEncoder);
        assertInstanceOf(BCryptPasswordEncoder.class, passwordEncoder);
    }

    @Test
    void testAuthenticationManagerBean() {
        var authenticationManager = securityConfig.authenticationManager();
        assertNotNull(authenticationManager);
        assertInstanceOf(AuthenticationManager.class, authenticationManager);
    }

    @Test
    void testCorsConfigurationSourceBean() {
        var corsConfigurationSource = securityConfig.corsConfigurationSource();
        assertNotNull(corsConfigurationSource);
        assertInstanceOf(CorsConfigurationSource.class, corsConfigurationSource);

        CorsConfiguration corsConfiguration = corsConfigurationSource.getCorsConfiguration(new MockHttpServletRequest());
        assertNotNull(corsConfiguration);
        assertNotNull(corsConfiguration.getAllowedOriginPatterns());
        assertNotNull(corsConfiguration.getAllowedMethods());
        assertNotNull(corsConfiguration.getAllowedHeaders());
        assertTrue(corsConfiguration.getAllowedOriginPatterns().contains("http://*:[*]"));
        assertTrue(corsConfiguration.getAllowedMethods().containsAll(List.of("GET", "POST", "PUT", "DELETE", "HEAD", "PATCH", "OPTIONS")));
        assertTrue(corsConfiguration.getAllowedHeaders().containsAll(List.of("Authorization", "Content-Type")));
    }

    @Test
    void testSecurityFilterChainBean() throws Exception {
        HttpSecurity httpSecurity = mock(HttpSecurity.class, RETURNS_DEEP_STUBS);

        SecurityFilterChain securityFilterChain = securityConfig.securityFilterChain(httpSecurity);

        assertNotNull(securityFilterChain);
    }
}