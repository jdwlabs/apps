package com.jdw.usersrole.controllers;

import com.jdw.usersrole.daos.ProfileIconDao;
import com.jdw.usersrole.repositories.ProfileRepository;
import com.jdw.usersrole.repositories.UserRepository;
import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.ProfileService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

// Standalone MockMvc does not evaluate @PreAuthorize, so these cover only the resource
// scoping applied underneath the authorization check. ProfileAddressAuthorizationTests
// covers the authorization check itself, where the interceptor actually runs.
@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class ProfileAddressScopeTests {
    private static final long OWNED_PROFILE_ID = 1L;
    private static final long OWNED_ADDRESS_ID = 10L;
    private static final long OTHER_PROFILE_ID = 2L;
    private static final long OTHER_ADDRESS_ID = 20L;

    @Mock
    private ProfileRepository profileRepository;
    @Mock
    private UserRepository userRepository;
    @Mock
    private ProfileIconDao profileIconDao;
    @Mock
    private JwtService jwtService;

    private Map<Long, Long> addressOwners;
    private MockMvc mockMvc;

    @BeforeEach
    void setUp() {
        addressOwners = new HashMap<>(Map.of(
                OWNED_ADDRESS_ID, OWNED_PROFILE_ID,
                OTHER_ADDRESS_ID, OTHER_PROFILE_ID));
        when(profileRepository.deleteAddress(anyLong(), anyLong())).thenAnswer(invocation -> {
            Long profileId = invocation.getArgument(0);
            Long addressId = invocation.getArgument(1);
            return addressOwners.remove(addressId, profileId);
        });
        when(jwtService.getEmailAddress(anyString())).thenReturn("user@jdw.com");
        ProfileService profileService = new ProfileService(profileRepository, userRepository, profileIconDao);
        mockMvc = MockMvcBuilders.standaloneSetup(new ProfilesController(profileService, jwtService))
                .setControllerAdvice(new GlobalExceptionHandler())
                .build();
    }

    @Test
    void deleteAddress_shouldReturnNotFoundAndKeepTheRow_whenAddressBelongsToAnotherProfile() throws Exception {
        mockMvc.perform(delete("/api/profiles/{profileId}/address/{addressId}", OWNED_PROFILE_ID, OTHER_ADDRESS_ID)
                        .header("Authorization", "Bearer token"))
                .andExpect(status().isNotFound());

        assertTrue(addressOwners.containsKey(OTHER_ADDRESS_ID));
        assertEquals(OTHER_PROFILE_ID, addressOwners.get(OTHER_ADDRESS_ID));
    }

    @Test
    void deleteAddress_shouldReturnNoContent_whenAddressBelongsToTheProfileInThePath() throws Exception {
        mockMvc.perform(delete("/api/profiles/{profileId}/address/{addressId}", OWNED_PROFILE_ID, OWNED_ADDRESS_ID)
                        .header("Authorization", "Bearer token"))
                .andExpect(status().isNoContent());

        assertFalse(addressOwners.containsKey(OWNED_ADDRESS_ID));
    }
}
