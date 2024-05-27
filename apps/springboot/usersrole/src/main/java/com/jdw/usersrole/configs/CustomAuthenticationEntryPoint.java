package com.jdw.usersrole.configs;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.web.AuthenticationEntryPoint;
import org.springframework.stereotype.Component;

import java.io.IOException;

@Component
@Slf4j
public class CustomAuthenticationEntryPoint implements AuthenticationEntryPoint{
    @Override
    public void commence(HttpServletRequest request, HttpServletResponse response, AuthenticationException authException) throws IOException {
        log.debug("Commencing custom authentication entry point");
        response.addHeader("Access-Denied-Reason", "Authentication Required");
        response.sendError(HttpServletResponse.SC_UNAUTHORIZED, "Access-Denied " + authException.getMessage());
    }
}
