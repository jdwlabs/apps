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
import org.springframework.security.access.AccessDeniedException;

import static org.junit.jupiter.api.Assertions.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class CustomAccessDeniedHandlerTests {
    @Mock
    private HttpServletRequest request;
    @InjectMocks
    private CustomAccessDeniedHandler customAccessDeniedHandler;

    @Test
    void handle() throws Exception {
        AccessDeniedException exception = new AccessDeniedException("Test Access Denied");
        MockHttpServletResponse mockResponse = new MockHttpServletResponse();

        customAccessDeniedHandler.handle(request, mockResponse, exception);

        assertEquals(HttpServletResponse.SC_FORBIDDEN, mockResponse.getStatus());
        assertEquals("Not Authorized", mockResponse.getHeader("Access-Denied-Reason"));
        assertEquals("Access Denied Test Access Denied", mockResponse.getErrorMessage());
    }
}