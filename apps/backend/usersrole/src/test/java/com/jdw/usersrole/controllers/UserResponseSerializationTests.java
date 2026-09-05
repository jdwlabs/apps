package com.jdw.usersrole.controllers;

import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.services.AuthService;
import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.ProfileService;
import com.jdw.usersrole.services.UserService;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import java.sql.Date;
import java.sql.Timestamp;
import java.util.List;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

// Standalone MockMvc serializes through Jackson's default converters rather than the
// context's. That is faithful here only because src/main registers no ObjectMapper bean
// to diverge from them; OpenApiPasswordContractTests covers the real context.
@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class UserResponseSerializationTests {
    private static final String BCRYPT_HASH = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy";

    @Mock
    private AuthService authService;
    @Mock
    private UserService userService;
    @Mock
    private JwtService jwtService;
    @Mock
    private ProfileService profileService;

    @Test
    void registrationResponse_shouldNotContainPasswordHash() throws Exception {
        when(userService.createUser(any())).thenReturn(userWithProfile());
        MockMvc mockMvc = MockMvcBuilders.standaloneSetup(new AuthController(authService, userService)).build();

        String body = mockMvc.perform(post("/auth/user")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"emailAddress\":\"user@jdw.com\",\"password\":\"P@ssw0rd!\"}"))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.password").doesNotExist())
                .andReturn().getResponse().getContentAsString();

        assertNoPasswordExposure(body);
    }

    @Test
    void usersPageResponse_shouldNotContainPasswordHash() throws Exception {
        when(userService.getAllUsers(0, 100)).thenReturn(List.of(userWithProfile()));
        MockMvc mockMvc = MockMvcBuilders.standaloneSetup(new UsersController(userService, jwtService)).build();

        String body = mockMvc.perform(get("/api/users"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].emailAddress").exists())
                .andExpect(jsonPath("$[0].password").doesNotExist())
                .andReturn().getResponse().getContentAsString();

        assertNoPasswordExposure(body);
    }

    @Test
    void singleUserResponse_shouldNotContainPasswordHash() throws Exception {
        when(userService.getUserById(1L)).thenReturn(userWithoutProfile());
        MockMvc mockMvc = MockMvcBuilders.standaloneSetup(new UsersController(userService, jwtService)).build();

        String body = mockMvc.perform(get("/api/users/1"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.password").doesNotExist())
                .andReturn().getResponse().getContentAsString();

        assertNoPasswordExposure(body);
    }

    @Test
    void userResponseWithEmbeddedProfile_shouldNotContainPasswordHash() throws Exception {
        when(userService.getUserByEmailAddress("user@jdw.com")).thenReturn(userWithProfile());
        MockMvc mockMvc = MockMvcBuilders.standaloneSetup(new UsersController(userService, jwtService)).build();

        String body = mockMvc.perform(get("/api/users/email/user@jdw.com"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.profile.firstName").value("Ada"))
                .andExpect(jsonPath("$.password").doesNotExist())
                .andReturn().getResponse().getContentAsString();

        assertNoPasswordExposure(body);
    }

    @Test
    void profileResponse_shouldNotContainPasswordHash() throws Exception {
        when(profileService.getProfileById(2L)).thenReturn(profile());
        MockMvc mockMvc = MockMvcBuilders.standaloneSetup(new ProfilesController(profileService, jwtService)).build();

        String body = mockMvc.perform(get("/api/profiles/2"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.firstName").value("Ada"))
                .andReturn().getResponse().getContentAsString();

        assertNoPasswordExposure(body);
    }

    private void assertNoPasswordExposure(String body) {
        assertFalse(body.contains("password"), "response body exposes a password key: " + body);
        assertFalse(body.contains("$2a$"), "response body exposes a bcrypt hash: " + body);
        assertFalse(body.contains("$2b$"), "response body exposes a bcrypt hash: " + body);
    }

    private User userWithoutProfile() {
        return baseUser().build();
    }

    private User userWithProfile() {
        return baseUser().profile(profile()).build();
    }

    private User.UserBuilder baseUser() {
        Timestamp now = new Timestamp(System.currentTimeMillis());
        return User.builder()
                .id(1L)
                .emailAddress("user@jdw.com")
                .password(BCRYPT_HASH)
                .status("ACTIVE")
                .roles(Set.of())
                .createdByUserId(1L)
                .createdTime(now)
                .modifiedByUserId(1L)
                .modifiedTime(now);
    }

    private Profile profile() {
        Timestamp now = new Timestamp(System.currentTimeMillis());
        return Profile.builder()
                .id(2L)
                .firstName("Ada")
                .lastName("Lovelace")
                .birthdate(Date.valueOf("1815-12-10"))
                .userId(1L)
                .addresses(Set.of())
                .createdByUserId(1L)
                .createdTime(now)
                .modifiedByUserId(1L)
                .modifiedTime(now)
                .build();
    }
}
