package com.jdw.usersrole.controllers;

import com.jdw.usersrole.models.User;
import com.jdw.usersrole.services.UserService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequiredArgsConstructor
@RequestMapping("/api/users")
@Slf4j
public class UsersController {
    private final UserService userService;

    @GetMapping
    public List<User> getAllUsers() {
        log.trace("Getting all users");
        return userService.getAllUsers();
    }

    @GetMapping("/{userId}")
    public User getUserById(@PathVariable Long userId) {
        log.trace("Getting user with id {}", userId);
        return userService.getUserById(userId);
    }

}
