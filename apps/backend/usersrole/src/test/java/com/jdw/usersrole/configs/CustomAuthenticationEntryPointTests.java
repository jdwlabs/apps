package com.jdw.usersrole.configs;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.mock.web.MockHttpServletResponse;
import org.springframework.security.core.AuthenticationException;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.mock;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class CustomAuthenticationEntryPointTests {
    @Mock
    private HttpServletRequest request;
    @Mock
    private AuthenticationException exception;
    @InjectMocks
    private CustomAuthenticationEntryPoint customAuthenticationEntryPoint;

    @Test
    void commence() throws Exception {
        MockHttpServletResponse mockResponse = new MockHttpServletResponse();
        exception = mock(AuthenticationException.class);
        mockResponse.addHeader("Access-Denied-Reason", "Authentication Required");

        customAuthenticationEntryPoint.commence(request, mockResponse, exception);

        assertEquals(HttpServletResponse.SC_UNAUTHORIZED, mockResponse.getStatus());
        assertEquals("Authentication Required", mockResponse.getHeader("Access-Denied-Reason"));
        assertEquals("Access Denied null", mockResponse.getErrorMessage());
    }
}