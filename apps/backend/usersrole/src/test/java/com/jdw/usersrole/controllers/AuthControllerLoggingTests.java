package com.jdw.usersrole.controllers;

import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import com.jdw.usersrole.dtos.AuthResponseDTO;
import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.services.AuthService;
import com.jdw.usersrole.services.UserService;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.slf4j.LoggerFactory;

import java.sql.Timestamp;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;

// Exercises the controller through the real Logback pipeline (not just UserRequestDTO in
// isolation) because the log call itself, not just the DTO, is the thing that must stay safe:
// a future call site could still hand the whole DTO to a formatter that ignores toString().
@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class AuthControllerLoggingTests {
    private static final String KNOWN_PASSWORD = "S3cr3t-Pass!23";

    @Mock
    private AuthService authService;
    @Mock
    private UserService userService;
    @InjectMocks
    private AuthController authController;

    private Logger controllerLogger;
    private Level originalLevel;
    private ListAppender<ILoggingEvent> appender;

    @BeforeEach
    void captureTraceLogsAtControllerLevel() {
        controllerLogger = (Logger) LoggerFactory.getLogger(AuthController.class);
        originalLevel = controllerLogger.getLevel();
        controllerLogger.setLevel(Level.TRACE);
        appender = new ListAppender<>();
        appender.start();
        controllerLogger.addAppender(appender);
    }

    @AfterEach
    void restoreLoggerState() {
        controllerLogger.detachAppender(appender);
        controllerLogger.setLevel(originalLevel);
    }

    @Test
    void authenticate_neverLogsTheCleartextPasswordAtTrace() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", KNOWN_PASSWORD);
        when(authService.authenticate(userRequestDTO))
                .thenReturn(AuthResponseDTO.builder().jwtToken("mock-jwt-token").build());

        authController.authenticate(userRequestDTO);

        List<String> messages = loggedMessages();
        assertFalse(messages.stream().anyMatch(message -> message.contains(KNOWN_PASSWORD)),
                "TRACE log output must never contain the cleartext password: " + messages);
    }

    @Test
    void createUser_neverLogsTheCleartextPasswordAtTrace() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", KNOWN_PASSWORD);
        when(userService.createUser(any(UserRequestDTO.class))).thenReturn(buildMockUser());

        authController.createUser(userRequestDTO);

        List<String> messages = loggedMessages();
        assertFalse(messages.stream().anyMatch(message -> message.contains(KNOWN_PASSWORD)),
                "TRACE log output must never contain the cleartext password: " + messages);
    }

    private List<String> loggedMessages() {
        return appender.list.stream().map(ILoggingEvent::getFormattedMessage).toList();
    }

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
}
