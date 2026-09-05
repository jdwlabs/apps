package com.jdw.usersrole.controllers;

import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.services.JwtService;
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

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class UsersControllerLoggingTests {
    private static final String KNOWN_PASSWORD = "S3cr3t-Pass!23";

    @Mock
    private UserService userService;
    @Mock
    private JwtService jwtService;
    @InjectMocks
    private UsersController usersController;

    private Logger controllerLogger;
    private Level originalLevel;
    private ListAppender<ILoggingEvent> appender;

    @BeforeEach
    void captureTraceLogsAtControllerLevel() {
        controllerLogger = (Logger) LoggerFactory.getLogger(UsersController.class);
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
    void createUser_neverLogsTheCleartextPasswordAtTrace() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", KNOWN_PASSWORD);
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("admin@jdw.com");
        when(userService.createUser(any(UserRequestDTO.class), any(String.class))).thenReturn(buildMockUser());

        usersController.createUser(userRequestDTO, "Bearer token");

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
