package com.jdw.usersrole.services;

import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.exceptions.ResourceExistsException;
import com.jdw.usersrole.exceptions.ResourceNotFoundException;
import com.jdw.usersrole.metrics.ExecutionTimeLogger;
import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.models.UserRole;
import com.jdw.usersrole.repositories.RoleRepository;
import com.jdw.usersrole.repositories.UserRepository;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.NotNull;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.security.access.AccessDeniedException;
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
    private final RoleRepository roleRepository;

    @ExecutionTimeLogger
    public List<User> getAllUsers() {
        log.info("Getting all users");
        return userRepository.findAll();
    }

    @ExecutionTimeLogger
    public User getUserById(@NotNull Long id) {
        log.info("Getting user with id: {}", id);
        return userRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("User not found with id " + id));
    }

    @ExecutionTimeLogger
    public User getUserByEmailAddress(@NotNull String emailAddress) {
        log.info("Getting user with email address: {}", emailAddress);
        return userRepository.findByEmailAddress(emailAddress)
                .orElseThrow(() -> new ResourceNotFoundException("User not found with email address " + emailAddress));
    }

    public User createUser(@NotNull @Valid UserRequestDTO userDTO) {
        log.info("Creating user: {}", userDTO);
        return createUser(userDTO, 1L);
    }

    public User createUser(@NotNull @Valid UserRequestDTO userDTO, @NotNull String emailAddress) {
        log.info("Creating user: user={}, requester={}", userDTO, emailAddress);
        Long userId = getUserIdByEmailAddress(emailAddress);
        return createUser(userDTO, userId);
    }

    @ExecutionTimeLogger
    public User createUser(@NotNull @Valid UserRequestDTO userDTO, @NotNull Long userId) {
        log.info("Creating user: user={}, requesterId={}", userDTO, userId);
        userRepository.findByEmailAddress(userDTO.emailAddress())
                .ifPresent(user -> {
                    throw new ResourceExistsException("User already exists with email address " + userDTO.emailAddress());
                });
        Timestamp currentTime = Timestamp.from(Instant.now());
        User newUser = User.builder()
                .emailAddress(userDTO.emailAddress())
                .password(passwordEncoder.encode(userDTO.password()))
                .status(Status.ACTIVE.name())
                .roles(null)
                .createdByUserId(userId)
                .createdTime(currentTime)
                .modifiedByUserId(userId)
                .modifiedTime(currentTime)
                .build();
        return userRepository.save(newUser);
    }

    @ExecutionTimeLogger
    public User updateUser(@NotNull Long id, @Valid UserRequestDTO userDTO, @NotNull String emailAddress) {
        log.info("Updating user: user={}, requester={}", userDTO, emailAddress);
        User currentUser = userRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("User not found with id " + id));
        Long userId = getUserIdByEmailAddress(emailAddress);
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
                .modifiedByUserId(userId)
                .modifiedTime(currentTime)
                .build();
        return userRepository.save(updatedUser);
    }

    @ExecutionTimeLogger
    public void deleteUser(@NotNull Long id, @NotNull String emailAddress) {
        log.info("Deleting user: id={}, requester={}", id, emailAddress);
        userRepository.deleteById(id);
    }

    @ExecutionTimeLogger
    public Long getUserIdByEmailAddress(@NotNull String emailAddress) {
        log.info("Getting user id with email address: {}", emailAddress);
        return getUserByEmailAddress(emailAddress).id();
    }

    @ExecutionTimeLogger
    public User grantRolesToUser(@NotNull Long id, @NotEmpty List<Long> roleIds, @NotNull String emailAddress) {
        log.info("Grating roles to user: id={}, roleIds={}, requester={}", id, roleIds, emailAddress);
        Long requesterUserId = getUserIdByEmailAddress(emailAddress);
        validateAdminRoleRequest(roleIds, requesterUserId);
        List<UserRole> userRoleList = buildUserRoleList(id, roleIds, requesterUserId);
        return userRepository.grantRoles(userRoleList);
    }

    @ExecutionTimeLogger
    public User revokeRolesFromUser(@NotNull Long id, @NotEmpty List<Long> roleIds, @NotNull String emailAddress) {
        log.info("Revoking roles from user: id={}, roleIds={}, requester={}", id, roleIds, emailAddress);
        Long requesterUserId = getUserIdByEmailAddress(emailAddress);
        validateAdminRoleRequest(roleIds, requesterUserId);
        List<UserRole> userRoleList = buildUserRoleList(id, roleIds, requesterUserId);
        return userRepository.revokeRoles(userRoleList);
    }

     List<UserRole> buildUserRoleList(Long userId, List<Long> roleIds, Long requesterUserId) {
        userRepository.findById(userId)
                .orElseThrow(() -> new ResourceNotFoundException("User not found with id " + userId));
        roleIds.forEach(roleId -> roleRepository.findById(roleId)
                .orElseThrow(() -> new ResourceNotFoundException("Role not found with id " + roleId)));
        Timestamp currentTime = Timestamp.from(Instant.now());
        return roleIds.stream()
                .map(roleId -> UserRole.builder()
                        .userId(userId)
                        .roleId(roleId)
                        .createdByUserId(requesterUserId)
                        .createdTime(currentTime)
                        .build())
                .toList();
    }

    void validateAdminRoleRequest(List<Long> roleIds, Long requesterUserId) {
        List<Long> elevatedRoleIds = List.of(1L);
        boolean containsElevatedRole = roleIds.stream().anyMatch(elevatedRoleIds::contains);

        if (containsElevatedRole) {
            boolean requesterHasElevatedRole = userRepository.hasAnyRole(requesterUserId, elevatedRoleIds);
            if (!requesterHasElevatedRole) {
                throw new AccessDeniedException("You do not have permission to grant or revoke elevated roles.");
            }
        }
    }
}
