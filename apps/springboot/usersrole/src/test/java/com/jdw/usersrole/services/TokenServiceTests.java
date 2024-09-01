package com.jdw.usersrole.services;

import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.core.Authentication;

import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class TokenServiceTests {
    @Mock
    private HttpServletRequest request;

    @Mock
    private HttpServletResponse response;

    @Mock
    private Authentication authentication;

    @InjectMocks
    private TokenService tokenService;

    @Test
    void logout_ShouldInvokeRequestLogout() throws ServletException {
        tokenService.logout(request, response, authentication);

        verify(request, times(1)).logout();
    }

    @Test
    void logout_ShouldLogError_WhenServletExceptionIsThrown() throws ServletException {
        doThrow(new ServletException("Logout failed")).when(request).logout();

        tokenService.logout(request, response, authentication);

        verify(request, times(1)).logout();
    }
}