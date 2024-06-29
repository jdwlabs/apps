package com.jdw.usersrole.services;

import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.exceptions.ResourceNotFoundException;
import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.repositories.UserRepository;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotNull;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class UserService {
    private final UserRepository userRepository;
    private final PasswordEncoder passwordEncoder;

    public List<User> getAllUsers() {
        log.info("Getting all users");
        return userRepository.findAll();
    }

    public User getUserById(@NotNull Long id) {
        log.info("Getting user with id: {}", id);
        return userRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("User not found with id " + id));
    }

    public User getUserByEmailAddress(@NotNull String emailAddress) {
        log.info("Getting user with email address: {}", emailAddress);
        return userRepository.findByEmailAddress(emailAddress)
                .orElseThrow(() -> new ResourceNotFoundException("User not found with email address " + emailAddress));
    }

    public User createUser(@Valid UserRequestDTO userDTO) {
        log.info("Creating user: {}", userDTO);
        Timestamp currentTime = Timestamp.from(Instant.now());
        User newUser = User.builder()
                .emailAddress(userDTO.emailAddress())
                .password(passwordEncoder.encode(userDTO.password()))
                .status(Status.ACTIVE.name())
                .roles(null)
                .createdByUserId(1L)
                .createdTime(currentTime)
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return userRepository.save(newUser);
    }

    public User updateUser(@NotNull Long id, @Valid UserRequestDTO userDTO) {
        log.info("Updating user: {}", userDTO);
        User currentUser = userRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("User not found with id " + id));
        Timestamp currentTime = Timestamp.from(Instant.now());
        User updatedUser = User.builder()
                .id(id)
                .emailAddress(userDTO.emailAddress())
                .password(passwordEncoder.encode(userDTO.password()))
                .status(currentUser.status())
                .roles(currentUser.roles())
                .profile(currentUser.profile())
                .createdByUserId(currentUser.createdByUserId())
                .createdTime(currentUser.createdTime())
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return userRepository.save(updatedUser);
    }

    public void deleteUser(@NotNull Long id) {
        log.info("Deleting user: {}", id);
        userRepository.deleteById(id);
    }
}
