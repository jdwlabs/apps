package com.jdw.usersrole.services;

import com.jdw.usersrole.dtos.UserRequestDTO;
import com.jdw.usersrole.exceptions.ResourceExistsException;
import com.jdw.usersrole.exceptions.ResourceNotFoundException;
import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.models.Status;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.repositories.RoleRepository;
import com.jdw.usersrole.repositories.UserRepository;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.access.AccessDeniedException;
import org.springframework.security.crypto.password.PasswordEncoder;

import java.util.List;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class UserServiceTests {
    @Mock
    private UserRepository userRepository;
    @Mock
    private PasswordEncoder passwordEncoder;
    @Mock
    private RoleRepository roleRepository;
    @InjectMocks
    private UserService userService;

    @Test
    void getAllUsers_ShouldReturnListOfUsers() {
        User user = User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .status(Status.ACTIVE.name())
                .build();
        when(userRepository.findAll()).thenReturn(List.of(user));

        List<User> users = userService.getAllUsers();

        assertNotNull(users);
        assertEquals(1, users.size());
        assertEquals("user@jdw.com", users.getFirst().emailAddress());
    }

    @Test
    void getUserById_ShouldReturnUser_WhenUserExists() {
        User user = User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .status(Status.ACTIVE.name())
                .build();
        when(userRepository.findById(1L)).thenReturn(Optional.of(user));

        User foundUser = userService.getUserById(1L);

        assertNotNull(foundUser);
        assertEquals(1L, foundUser.id());
        assertEquals("user@jdw.com", foundUser.emailAddress());
    }

    @Test
    void getUserById_ShouldThrowException_WhenUserNotFound() {
        when(userRepository.findById(anyLong())).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> userService.getUserById(1L));

        assertEquals("User not found with id 1", exception.getMessage());
    }

    @Test
    void getUserByEmailAddress_ShouldReturnUser_WhenUserExists() {
        User user = User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .status(Status.ACTIVE.name())
                .build();
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));

        User foundUser = userService.getUserByEmailAddress("user@jdw.com");

        assertNotNull(foundUser);
        assertEquals(1L, foundUser.id());
        assertEquals("user@jdw.com", foundUser.emailAddress());
    }

    @Test
    void getUserByEmailAddress_ShouldThrowException_WhenUserNotFound() {
        when(userRepository.findByEmailAddress(anyString())).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> userService.getUserByEmailAddress("user@jdw.com"));

        assertEquals("User not found with email address user@jdw.com", exception.getMessage());
    }

    @Test
    void createUser_ShouldCreateAndReturnUser() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", "Password1!");
        when(userRepository.findByEmailAddress(anyString())).thenReturn(Optional.empty());
        when(passwordEncoder.encode(anyString())).thenReturn("encodedPassword");
        when(userRepository.save(any(User.class))).thenAnswer(invocation -> invocation.getArgument(0));

        User createdUser = userService.createUser(userRequestDTO);

        assertNotNull(createdUser);
        assertEquals("user@jdw.com", createdUser.emailAddress());
        assertEquals("encodedPassword", createdUser.password());
        assertEquals(Status.ACTIVE.name(), createdUser.status());
        verify(userRepository, times(1)).save(any(User.class));
    }

    @Test
    void createUser_ShouldThrowException_WhenUserAlreadyExists() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", "Password1!");
        when(userRepository.findByEmailAddress(anyString())).thenReturn(Optional.of(User.builder().build()));

        ResourceExistsException exception = assertThrows(ResourceExistsException.class,
                () -> userService.createUser(userRequestDTO));

        assertEquals("User already exists with email address user@jdw.com", exception.getMessage());
    }

    @Test
    void createUser_ShouldDelegateToCreateUserWithUserId() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", "Password1!");
        String requesterEmail = "requester@jdw.com";

        when(userRepository.findByEmailAddress(userRequestDTO.emailAddress())).thenReturn(Optional.empty());
        when(userRepository.findByEmailAddress(requesterEmail)).thenReturn(Optional.of(User.builder().id(2L).build()));
        when(passwordEncoder.encode(userRequestDTO.password())).thenReturn("encodedPassword");
        when(userRepository.save(any(User.class))).thenAnswer(invocation -> invocation.getArgument(0));

        User createdUser = userService.createUser(userRequestDTO, requesterEmail);

        assertNotNull(createdUser);
        assertEquals("user@jdw.com", createdUser.emailAddress());
        assertEquals("encodedPassword", createdUser.password());
        assertEquals(Status.ACTIVE.name(), createdUser.status());
        verify(userRepository, times(1)).save(any(User.class));
    }

    @Test
    void updateUser_ShouldUpdateAndReturnUser() {
        User existingUser = User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .password("oldPassword")
                .status(Status.ACTIVE.name())
                .build();
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", "NewPassword1!");

        when(userRepository.findById(1L)).thenReturn(Optional.of(existingUser));
        when(userRepository.findByEmailAddress("requester@jdw.com")).thenReturn(Optional.of(User.builder().id(1L).build()));
        when(passwordEncoder.encode(anyString())).thenReturn("encodedNewPassword");
        when(userRepository.save(any(User.class))).thenAnswer(invocation -> invocation.getArgument(0));

        User updatedUser = userService.updateUser(1L, userRequestDTO, "requester@jdw.com");

        assertNotNull(updatedUser);
        assertEquals("user@jdw.com", updatedUser.emailAddress());
        assertEquals("encodedNewPassword", updatedUser.password());
        assertEquals(existingUser.status(), updatedUser.status());
        verify(userRepository, times(1)).save(any(User.class));
    }

    @Test
    void updateUser_ShouldThrowException_WhenUserNotFound() {
        UserRequestDTO userRequestDTO = new UserRequestDTO("user@jdw.com", "NewPassword1!");
        when(userRepository.findById(anyLong())).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> userService.updateUser(1L, userRequestDTO, "requester@jdw.com"));

        assertEquals("User not found with id 1", exception.getMessage());
    }

    @Test
    void deleteUser_ShouldDeleteUser_WhenUserExists() {
        doNothing().when(userRepository).deleteById(anyLong());

        userService.deleteUser(1L, "requester@jdw.com");

        verify(userRepository, times(1)).deleteById(1L);
    }

    @Test
    void getUserIdByEmailAddress_ShouldReturnUserId_WhenUserExists() {
        User user = User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .build();
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));

        Long userId = userService.getUserIdByEmailAddress("user@jdw.com");

        assertNotNull(userId);
        assertEquals(1L, userId);
    }

    @Test
    void getUserIdByEmailAddress_ShouldThrowException_WhenUserNotFound() {
        when(userRepository.findByEmailAddress(anyString())).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> userService.getUserIdByEmailAddress("user@jdw.com"));

        assertEquals("User not found with email address user@jdw.com", exception.getMessage());
    }

    @Test
    void grantRolesToUser_ShouldReturnUpdatedUser() {
        Long userId = 1L;
        List<Long> roleIds = List.of(2L, 3L);
        String emailAddress = "requester@jdw.com";
        User existingUser = User.builder()
                .id(userId)
                .emailAddress("user@jdw.com")
                .build();
        User updatedUser = User.builder()
                .id(userId)
                .emailAddress("user@jdw.com")
                .build();

        when(userRepository.findByEmailAddress(emailAddress)).thenReturn(Optional.of(User.builder().build()));
        when(userRepository.findById(userId)).thenReturn(Optional.of(existingUser));
        when(roleRepository.findById(anyLong())).thenReturn(Optional.of(Role.builder().build()));
        when(userRepository.grantRoles(any())).thenReturn(updatedUser);

        User result = userService.grantRolesToUser(userId, roleIds, emailAddress);

        assertNotNull(result);
        assertEquals(userId, result.id());
        verify(userRepository).grantRoles(anyList());
    }

    @Test
    void grantRolesToUser_ShouldThrowException_WhenUserNotFound() {
        Long userId = 1L;
        List<Long> roleIds = List.of(2L, 3L);
        String emailAddress = "requester@jdw.com";

        when(userRepository.findByEmailAddress(emailAddress)).thenReturn(Optional.of(User.builder().build()));
        when(userRepository.findById(userId)).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> userService.grantRolesToUser(userId, roleIds, emailAddress));

        assertEquals("User not found with id 1", exception.getMessage());
    }

    @Test
    void grantRolesToUser_ShouldThrowException_WhenRoleNotFound() {
        Long userId = 1L;
        List<Long> roleIds = List.of(2L, 3L);
        String emailAddress = "requester@jdw.com";

        when(userRepository.findByEmailAddress(emailAddress)).thenReturn(Optional.of(User.builder().build()));
        when(userRepository.findById(userId)).thenReturn(Optional.of(User.builder().id(userId).build()));
        when(roleRepository.findById(2L)).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> userService.grantRolesToUser(userId, roleIds, emailAddress));

        assertEquals("Role not found with id 2", exception.getMessage());
    }

    @Test
    void revokeRolesFromUser_ShouldReturnUpdatedUser() {
        Long userId = 1L;
        List<Long> roleIds = List.of(2L, 3L);
        String emailAddress = "requester@jdw.com";
        User existingUser = User.builder()
                .id(userId)
                .emailAddress("user@jdw.com")
                .build();
        User updatedUser = User.builder()
                .id(userId)
                .emailAddress("user@jdw.com")
                .build();

        when(userRepository.findByEmailAddress(emailAddress)).thenReturn(Optional.of(User.builder().build()));
        when(userRepository.findById(userId)).thenReturn(Optional.of(existingUser));
        when(roleRepository.findById(anyLong())).thenReturn(Optional.of(Role.builder().build()));
        when(userRepository.revokeRoles(any())).thenReturn(updatedUser);

        User result = userService.revokeRolesFromUser(userId, roleIds, emailAddress);

        assertNotNull(result);
        assertEquals(userId, result.id());
        verify(userRepository).revokeRoles(anyList());
    }

    @Test
    void revokeRolesFromUser_ShouldThrowException_WhenUserNotFound() {
        Long userId = 1L;
        List<Long> roleIds = List.of(2L, 3L);
        String emailAddress = "requester@jdw.com";

        when(userRepository.findByEmailAddress(emailAddress)).thenReturn(Optional.of(User.builder().build()));
        when(userRepository.findById(userId)).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> userService.revokeRolesFromUser(userId, roleIds, emailAddress));

        assertEquals("User not found with id 1", exception.getMessage());
    }

    @Test
    void revokeRolesFromUser_ShouldThrowException_WhenRoleNotFound() {
        Long userId = 1L;
        List<Long> roleIds = List.of(2L, 3L);
        String emailAddress = "requester@jdw.com";

        when(userRepository.findByEmailAddress(emailAddress)).thenReturn(Optional.of(User.builder().build()));
        when(userRepository.findById(userId)).thenReturn(Optional.of(User.builder().id(userId).build()));
        when(roleRepository.findById(2L)).thenReturn(Optional.empty());

        ResourceNotFoundException exception = assertThrows(ResourceNotFoundException.class,
                () -> userService.revokeRolesFromUser(userId, roleIds, emailAddress));

        assertEquals("Role not found with id 2", exception.getMessage());
    }

    @Test
    void validateAdminRoleRequest_ShouldNotThrowException_WhenRequesterHasElevatedRole() {
        Long requesterUserId = 2L;
        List<Long> roleIds = List.of(1L, 3L); // Contains elevated role (1L)
        List<Long> elevatedRoleIds = List.of(1L);

        when(userRepository.hasAnyRole(requesterUserId, elevatedRoleIds)).thenReturn(true);

        assertDoesNotThrow(() -> userService.validateAdminRoleRequest(roleIds, requesterUserId));
    }

    @Test
    void validateAdminRoleRequest_ShouldThrowAccessDeniedException_WhenRequesterDoesNotHaveElevatedRole() {
        Long requesterUserId = 2L;
        List<Long> roleIds = List.of(1L, 3L); // Contains elevated role (1L)
        List<Long> elevatedRoleIds = List.of(1L);

        when(userRepository.hasAnyRole(requesterUserId, elevatedRoleIds)).thenReturn(false);

        AccessDeniedException exception = assertThrows(AccessDeniedException.class,
                () -> userService.validateAdminRoleRequest(roleIds, requesterUserId));

        assertEquals("You do not have permission to grant or revoke elevated roles.", exception.getMessage());
    }

    @Test
    void validateAdminRoleRequest_ShouldNotThrowException_WhenNoElevatedRoleInRoleIds() {
        Long requesterUserId = 2L;
        List<Long> roleIds = List.of(2L, 3L); // No elevated role

        assertDoesNotThrow(() -> userService.validateAdminRoleRequest(roleIds, requesterUserId));
    }
}