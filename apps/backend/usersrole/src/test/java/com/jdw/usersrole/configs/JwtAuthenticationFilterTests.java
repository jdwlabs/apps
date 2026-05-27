package com.jdw.usersrole.configs;

import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.JwtUserDetailService;
import io.jsonwebtoken.JwtException;
import jakarta.servlet.FilterChain;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.context.SecurityContext;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.core.userdetails.UserDetails;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class JwtAuthenticationFilterTests {
    @Mock
    private JwtUserDetailService jwtUserDetailService;
    @Mock
    private JwtService jwtService;
    @Mock
    private HttpServletRequest request;
    @Mock
    private HttpServletResponse response;
    @Mock
    private FilterChain chain;
    @InjectMocks
    private JwtAuthenticationFilter jwtAuthenticationFilter;

    @BeforeEach
    void setUp() {
        SecurityContextHolder.setContext(mock(SecurityContext.class));
    }

    @Test
    void doFilterInternal_whenAuthorizationHeaderIsValid_setsAuthentication() throws Exception {
        String authHeader = "Bearer validToken";
        String emailAddress = "user@jdw.com";
        String token = "validToken";
        UserDetails userDetails = mock(UserDetails.class);

        when(request.getHeader("Authorization")).thenReturn(authHeader);
        when(jwtService.getEmailAddress(authHeader)).thenReturn(emailAddress);
        when(jwtUserDetailService.loadUserByUsername(emailAddress)).thenReturn(userDetails);
        when(jwtService.getJwtToken(authHeader)).thenReturn(token);
        when(jwtService.isTokenValid(token, userDetails)).thenReturn(true);

        jwtAuthenticationFilter.doFilterInternal(request, response, chain);

        verify(chain).doFilter(request, response);
        verify(SecurityContextHolder.getContext()).setAuthentication(any(UsernamePasswordAuthenticationToken.class));
    }

    @Test
    void doFilterInternal_whenAuthorizationHeaderIsInvalid_doesNotSetAuthentication() throws Exception {
        when(request.getHeader("Authorization")).thenReturn("InvalidHeader");

        jwtAuthenticationFilter.doFilterInternal(request, response, chain);

        verify(chain).doFilter(request, response);
        verify(SecurityContextHolder.getContext(), never()).setAuthentication(any(UsernamePasswordAuthenticationToken.class));
    }

    @Test
    void doFilterInternal_whenAuthorizationHeaderIsNull_doesNotSetAuthentication() throws Exception {
        when(request.getHeader("Authorization")).thenReturn(null);

        jwtAuthenticationFilter.doFilterInternal(request, response, chain);

        verify(chain).doFilter(request, response);
        verify(SecurityContextHolder.getContext(), never()).setAuthentication(any(UsernamePasswordAuthenticationToken.class));
    }

    @Test
    void doFilterInternal_whenEmailAddressIsNull_doesNotSetAuthentication() throws Exception {
        String authHeader = "Bearer validToken";

        when(request.getHeader("Authorization")).thenReturn(authHeader);
        when(jwtService.getEmailAddress(authHeader)).thenReturn(null);

        jwtAuthenticationFilter.doFilterInternal(request, response, chain);

        verify(chain).doFilter(request, response);
        verify(SecurityContextHolder.getContext(), never()).setAuthentication(any(UsernamePasswordAuthenticationToken.class));
    }

    @Test
    void doFilterInternal_whenAuthenticationAlreadySet_doesNotModifySecurityContext() throws Exception {
        String authHeader = "Bearer validToken";
        String emailAddress = "user@jdw.com";
        UsernamePasswordAuthenticationToken existingAuth = mock(UsernamePasswordAuthenticationToken.class);
        SecurityContext securityContext = mock(SecurityContext.class);
        when(securityContext.getAuthentication()).thenReturn(existingAuth);
        SecurityContextHolder.setContext(securityContext);
        when(request.getHeader("Authorization")).thenReturn(authHeader);
        when(jwtService.getEmailAddress(authHeader)).thenReturn(emailAddress);

        jwtAuthenticationFilter.doFilterInternal(request, response, chain);

        verify(chain).doFilter(request, response);
        verify(jwtUserDetailService, never()).loadUserByUsername(emailAddress);
        verify(securityContext, never()).setAuthentication(any(UsernamePasswordAuthenticationToken.class));
    }

    @Test
    void doFilterInternal_whenJwtExceptionOccurs_returnsUnauthorized() throws Exception {
        String authHeader = "Bearer invalidToken";

        when(request.getHeader("Authorization")).thenReturn(authHeader);
        when(jwtService.getEmailAddress(authHeader)).thenThrow(new JwtException("Invalid token"));

        jwtAuthenticationFilter.doFilterInternal(request, response, chain);

        verify(chain).doFilter(request, response);
        verify(response).setStatus(HttpServletResponse.SC_UNAUTHORIZED);
    }

    @Test
    void doFilterInternal_whenJwtTokenIsExpired_doesNotSetAuthentication() throws Exception {
        String authHeader = "Bearer validToken";
        String emailAddress = "user@jdw.com";
        String token = "validToken";
        UserDetails userDetails = mock(UserDetails.class);

        when(request.getHeader("Authorization")).thenReturn(authHeader);
        when(jwtService.getEmailAddress(authHeader)).thenReturn(emailAddress);
        when(jwtUserDetailService.loadUserByUsername(emailAddress)).thenReturn(userDetails);
        when(jwtService.getJwtToken(authHeader)).thenReturn(token);
        when(jwtService.isTokenValid(token, userDetails)).thenReturn(false);

        jwtAuthenticationFilter.doFilterInternal(request, response, chain);

        verify(chain).doFilter(request, response);
        verify(SecurityContextHolder.getContext(), never()).setAuthentication(any(UsernamePasswordAuthenticationToken.class));
    }

    @Test
    void shouldNotFilterErrorDispatch() {
        boolean result = jwtAuthenticationFilter.shouldNotFilterErrorDispatch();

        assertFalse(result);
    }

    @Test
    void doFilterInternal_whenRequestIsNull_throwsNullPointerException() {
        assertThrows(NullPointerException.class, () ->
                jwtAuthenticationFilter.doFilterInternal(null, response, chain));
    }

    @Test
    void doFilterInternal_whenResponseIsNull_throwsNullPointerException() {
        assertThrows(NullPointerException.class, () ->
                jwtAuthenticationFilter.doFilterInternal(request, null, chain));
    }

    @Test
    void doFilterInternal_whenFilterChainIsNull_throwsNullPointerException() {
        assertThrows(NullPointerException.class, () ->
                jwtAuthenticationFilter.doFilterInternal(request, response, null));
    }
}