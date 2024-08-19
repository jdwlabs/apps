package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.services.AuthService;
import com.jdw.usersrole.services.UserService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequiredArgsConstructor
@RequestMapping("/auth")
@Slf4j
public class AuthController {
    private final AuthService authService;
    private final UserService userService;

    @PostMapping("/authenticate")
    public ResponseEntity<String> authenticate(@Valid @RequestBody UserRequestDTO user) {
        log.trace("Authenticating user {}", user);
        String token = authService.authenticate(user);
        return ResponseEntity.ok(token);
    }

    @PostMapping("/user")
    public ResponseEntity<User> createUser(@Valid @RequestBody UserRequestDTO user) {
        log.trace("Creating user {}", user);
        User userCreated = userService.createUser(user);
        return ResponseEntity.status(HttpStatus.CREATED).body(userCreated);
    }
}
