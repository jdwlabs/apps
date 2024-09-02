package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.UserService;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotEmpty;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequiredArgsConstructor
@RequestMapping("/api/users")
@Slf4j
public class UsersController {
    private final UserService userService;
    private final JwtService jwtService;

    @PreAuthorize("hasAuthority('ADMIN')")
    @GetMapping
    public ResponseEntity<List<User>> getAllUsers() {
        log.trace("Getting all users");
        return ResponseEntity.ok(userService.getAllUsers());
    }

    @PreAuthorize("hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()")
    @GetMapping("/{userId}")
    public ResponseEntity<User> getUserById(@PathVariable Long userId) {
        log.trace("Getting user with id {}", userId);
        User user = userService.getUserById(userId);
        return ResponseEntity.ok(user);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #emailAddress == authentication.principal.getUsername()")
    @GetMapping("/email/{emailAddress}")
    public ResponseEntity<User> getUserByEmailAddress(@PathVariable String emailAddress) {
        log.trace("Getting user with email address {}", emailAddress);
        User user = userService.getUserByEmailAddress(emailAddress);
        return ResponseEntity.ok(user);
    }

    @PostMapping
    public ResponseEntity<User> createUser(@Valid @RequestBody UserRequestDTO user, @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Creating user {}", user);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        User userCreated = userService.createUser(user, emailAddress);
        return ResponseEntity.status(HttpStatus.CREATED).body(userCreated);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()")
    @PutMapping("/{userId}")
    public ResponseEntity<User> updateUser(@PathVariable Long userId, @Valid @RequestBody UserRequestDTO user, @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Updating user with id {}", userId);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        User userUpdated = userService.updateUser(userId, user, emailAddress);
        return ResponseEntity.ok(userUpdated);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()")
    @DeleteMapping("/{userId}")
    public ResponseEntity<User> deleteUser(@PathVariable Long userId, @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Deleting user with id {}", userId);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        userService.deleteUser(userId, emailAddress);
        return ResponseEntity.noContent().build();
    }

    @PreAuthorize("hasAnyAuthority('ADMIN', 'MANAGER')")
    @PutMapping("/{userId}/roles/grant")
    public ResponseEntity<User> grantRolesToUser(
            @PathVariable Long userId,
            @NotEmpty @RequestBody List<Long> roleIds,
            @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Granting roles {} to user with id {}", roleIds, userId);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        User updatedUser = userService.grantRolesToUser(userId, roleIds, emailAddress);
        return ResponseEntity.ok(updatedUser);
    }

    @PreAuthorize("hasAnyAuthority('ADMIN', 'MANAGER')")
    @PutMapping("/{userId}/roles/revoke")
    public ResponseEntity<User> revokeRolesFromUser(
            @PathVariable Long userId,
            @NotEmpty @RequestBody List<Long> roleIds,
            @RequestHeader(name = "Authorization") String authorizationHeader) {
        log.trace("Revoking roles {} from user with id {}", roleIds, userId);
        String emailAddress = jwtService.getEmailAddress(authorizationHeader);
        User updatedUser = userService.revokeRolesFromUser(userId, roleIds, emailAddress);
        return ResponseEntity.ok(updatedUser);
    }
}
