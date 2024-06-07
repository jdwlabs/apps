package com.jdw.usersrole.services;

import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.repositories.UserRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class UserService {
    private final UserRepository userRepository;

    public List<User> getAllUsers() {
        log.info("Getting all users");
        return userRepository.findAll();
    }

    public User getUserById(Long id) {
        log.info("Getting user by id: {}", id);
        return userRepository.findById(id).orElse(null);
    }


    public User getUserByEmailAddress(String emailAddress) {
        log.info("Getting user by email address: {}", emailAddress);
        return userRepository.findByEmailAddress(emailAddress).orElse(null);
    }

    public User createUser(User user) {
        log.info("Creating user: {}", user);
        Timestamp currentTime = Timestamp.from(Instant.now());
        User newUser = User.builder()
                .emailAddress(user.emailAddress())
                .password(user.password())
                .status(Status.ACTIVE.name())
                .roles(null)
                .createdByUserId(1L)
                .createdTime(currentTime)
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return userRepository.save(newUser);
    }

    public User updateUser(Long id, User user) {
        log.info("Updating user: {}", user);
        if (user == null
                || id == null
                || user.emailAddress() == null
                || user.password() == null
                || !userRepository.existsById(id)) {
            return null;
        }
        User currentUser = userRepository.findById(id).orElse(null);
        if (currentUser == null) {
            return null;
        }
        String status = user.status() != null ? user.status() : Status.ACTIVE.name();
        Timestamp currentTime = Timestamp.from(Instant.now());
        User updatedUser = User.builder()
                .id(id)
                .emailAddress(user.emailAddress())
                .password(user.password())
                .status(status)
                .roles(currentUser.roles())
                .profile(currentUser.profile())
                .createdByUserId(currentUser.createdByUserId())
                .createdTime(currentUser.createdTime())
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return userRepository.save(updatedUser);
    }

    public void deleteUser(Long id) {
        log.info("Deleting user: {}", id);
        userRepository.deleteById(id);
    }

}
