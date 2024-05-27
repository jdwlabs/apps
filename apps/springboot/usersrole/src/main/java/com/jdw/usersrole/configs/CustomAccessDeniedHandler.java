package com.jdw.usersrole.configs;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.security.access.AccessDeniedException;
import org.springframework.security.web.access.AccessDeniedHandler;
import org.springframework.stereotype.Component;

import java.io.IOException;

@Component
@Slf4j
public class CustomAccessDeniedHandler implements AccessDeniedHandler {
    @Override
    public void handle(HttpServletRequest request, HttpServletResponse response, AccessDeniedException accessDeniedException) throws IOException {
        log.debug("Handling custom access denied handler");
        response.addHeader("Access-Denied-Reason", "Not Authorized");
        response.sendError(HttpServletResponse.SC_FORBIDDEN, "Access Denied " + accessDeniedException.getMessage());
    }
}
