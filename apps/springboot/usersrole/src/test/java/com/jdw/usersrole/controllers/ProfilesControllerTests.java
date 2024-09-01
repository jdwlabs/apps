package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.AddressRequestDTO;
import com.jdw.usersrole.dtos.ProfileCreateRequestDTO;
import com.jdw.usersrole.dtos.ProfileUpdateRequestDTO;
import com.jdw.usersrole.models.Address;
import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.models.ProfileIcon;
import com.jdw.usersrole.services.JwtService;
import com.jdw.usersrole.services.ProfileService;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.multipart.MultipartFile;

import java.sql.Timestamp;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class ProfilesControllerTests {
    @Mock
    private ProfileService profileService;
    @Mock
    private JwtService jwtService;
    @InjectMocks
    private ProfilesController profilesController;

    private Profile buildMockProfile() {
        Address address = Address.builder()
                .id(1L)
                .addressLine1("123 Street")
                .addressLine2("Apt 4B")
                .city("Cityville")
                .stateProvince("Stateville")
                .postalCode("12345")
                .country("Countryland")
                .profileId(1L)
                .createdByUserId(1L)
                .createdTime(new Timestamp(System.currentTimeMillis()))
                .modifiedByUserId(1L)
                .modifiedTime(new Timestamp(System.currentTimeMillis()))
                .build();

        ProfileIcon icon = ProfileIcon.builder()
                .id(1L)
                .profileId(1L)
                .icon(new byte[]{1, 2, 3})
                .createdByUserId(1L)
                .createdTime(new Timestamp(System.currentTimeMillis()))
                .modifiedByUserId(1L)
                .modifiedTime(new Timestamp(System.currentTimeMillis()))
                .build();

        return Profile.builder()
                .id(1L)
                .firstName("John")
                .middleName("M")
                .lastName("Doe")
                .birthdate(new java.sql.Date(System.currentTimeMillis()))
                .userId(1L)
                .addresses(new HashSet<>(Set.of(address)))
                .icon(icon)
                .createdByUserId(1L)
                .createdTime(new Timestamp(System.currentTimeMillis()))
                .modifiedByUserId(1L)
                .modifiedTime(new Timestamp(System.currentTimeMillis()))
                .build();
    }

    @Test
    void getProfiles_shouldReturnProfilesList() {
        Profile mockProfile = buildMockProfile();
        when(profileService.getProfiles()).thenReturn(List.of(mockProfile));

        ResponseEntity<List<Profile>> response = profilesController.getProfiles();

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertNotNull(response.getBody());
        assertEquals(1, response.getBody().size());
        assertEquals(mockProfile, response.getBody().getFirst());
    }

    @Test
    void getProfileById_shouldReturnProfile() {
        Long profileId = 1L;
        Profile mockProfile = buildMockProfile();
        when(profileService.getProfileById(profileId)).thenReturn(mockProfile);

        ResponseEntity<Profile> response = profilesController.getProfileById(profileId);

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(mockProfile, response.getBody());
    }

    @Test
    void getProfileByUserId_shouldReturnProfile() {
        Long userId = 1L;
        Profile mockProfile = buildMockProfile();
        when(profileService.getProfileByUserId(userId)).thenReturn(mockProfile);

        ResponseEntity<Profile> response = profilesController.getProfileByUserId(userId);

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(mockProfile, response.getBody());
    }

    @Test
    void createProfile_shouldReturnCreatedProfile() {
        ProfileCreateRequestDTO request = new ProfileCreateRequestDTO("John", "M", "Doe", new java.sql.Date(System.currentTimeMillis()), 1L);
        Profile createdProfile = buildMockProfile();
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(profileService.createProfile(any(ProfileCreateRequestDTO.class), any(String.class))).thenReturn(createdProfile);

        ResponseEntity<Profile> response = profilesController.createProfile(request, "Bearer token");

        assertEquals(HttpStatus.CREATED, response.getStatusCode());
        assertEquals(createdProfile, response.getBody());
    }

    @Test
    void updateProfileById_shouldReturnUpdatedProfile() {
        Long profileId = 1L;
        ProfileUpdateRequestDTO request = new ProfileUpdateRequestDTO("John", "M", "Doe", new java.sql.Date(System.currentTimeMillis()));
        Profile updatedProfile = buildMockProfile();
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(profileService.updateProfileById(any(Long.class), any(ProfileUpdateRequestDTO.class), any(String.class))).thenReturn(updatedProfile);

        ResponseEntity<Profile> response = profilesController.updateProfileById(profileId, request, "Bearer token");

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(updatedProfile, response.getBody());
    }

    @Test
    void updateProfileByUserId_shouldReturnUpdatedProfile() {
        Long userId = 1L;
        ProfileUpdateRequestDTO request = new ProfileUpdateRequestDTO("John", "M", "Doe", new java.sql.Date(System.currentTimeMillis()));
        Profile updatedProfile = buildMockProfile();
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(profileService.updateProfileByUserId(any(Long.class), any(ProfileUpdateRequestDTO.class), any(String.class))).thenReturn(updatedProfile);

        ResponseEntity<Profile> response = profilesController.updateProfileByUserId(userId, request, "Bearer token");

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(updatedProfile, response.getBody());
    }

    @Test
    void deleteProfileById_shouldReturnNoContent() {
        Long profileId = 1L;
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");

        ResponseEntity<Profile> response = profilesController.deleteProfileById(profileId, "Bearer token");

        assertEquals(HttpStatus.NO_CONTENT, response.getStatusCode());
    }

    @Test
    void deleteProfileByUserId_shouldReturnNoContent() {
        Long userId = 1L;
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");

        ResponseEntity<Profile> response = profilesController.deleteProfileByUserId(userId, "Bearer token");

        assertEquals(HttpStatus.NO_CONTENT, response.getStatusCode());
    }

    @Test
    void addAddress_shouldReturnUpdatedProfile() {
        Long profileId = 1L;
        AddressRequestDTO addressRequest = new AddressRequestDTO("123 Street", "Apt 4B", "Cityville", "Stateville", "12345", "Countryland");
        Profile updatedProfile = mock(Profile.class);
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(profileService.addAddress(profileId, addressRequest, "user@jdw.com")).thenReturn(updatedProfile);

        ResponseEntity<Profile> response = profilesController.addAddress(profileId, addressRequest, "Bearer token");

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(updatedProfile, response.getBody());
        verify(profileService).addAddress(profileId, addressRequest, "user@jdw.com");
    }

    @Test
    void updateAddress_shouldReturnUpdatedProfile() {
        Long profileId = 1L;
        Long addressId = 1L;
        AddressRequestDTO addressRequest = new AddressRequestDTO("123 Street", "Apt 4B", "Cityville", "Stateville", "12345", "Countryland");
        Profile updatedProfile = mock(Profile.class);
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(profileService.updateAddress(profileId, addressId, addressRequest, "user@jdw.com")).thenReturn(updatedProfile);

        ResponseEntity<Profile> response = profilesController.updateAddress(profileId, addressId, addressRequest, "Bearer token");

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(updatedProfile, response.getBody());
        verify(profileService).updateAddress(profileId, addressId, addressRequest, "user@jdw.com");
    }

    @Test
    void deleteAddress_shouldReturnNoContent() {
        Long profileId = 1L;
        Long addressId = 1L;
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");

        ResponseEntity<Profile> response = profilesController.deleteAddress(profileId, addressId, "Bearer token");

        assertEquals(HttpStatus.NO_CONTENT, response.getStatusCode());
        verify(profileService).deleteAddress(profileId, addressId, "user@jdw.com");
    }

    @Test
    void getProfileIcon_shouldReturnIcon() {
        Long profileId = 1L;
        byte[] mockIcon = new byte[]{1, 2, 3};
        when(profileService.getIcon(profileId)).thenReturn(mockIcon);

        ResponseEntity<byte[]> response = profilesController.getProfileIcon(profileId);

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(mockIcon, response.getBody());
    }

    @Test
    void addIcon_shouldReturnUpdatedProfile() {
        Long profileId = 1L;
        MultipartFile mockFile = mock(MultipartFile.class);
        Profile updatedProfile = mock(Profile.class);
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(profileService.addIcon(profileId, mockFile, "user@jdw.com")).thenReturn(updatedProfile);

        ResponseEntity<Profile> response = profilesController.addIcon(profileId, mockFile, "Bearer token");

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(updatedProfile, response.getBody());
        verify(profileService).addIcon(profileId, mockFile, "user@jdw.com");
    }

    @Test
    void updateIcon_shouldReturnUpdatedProfile() {
        Long profileId = 1L;
        MultipartFile mockFile = mock(MultipartFile.class);
        Profile updatedProfile = mock(Profile.class);
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");
        when(profileService.updateIcon(profileId, mockFile, "user@jdw.com")).thenReturn(updatedProfile);

        ResponseEntity<Profile> response = profilesController.updateIcon(profileId, mockFile, "Bearer token");

        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(updatedProfile, response.getBody());
        verify(profileService).updateIcon(profileId, mockFile, "user@jdw.com");
    }

    @Test
    void deleteIcon_shouldReturnNoContent() {
        Long profileId = 1L;
        when(jwtService.getEmailAddress(any(String.class))).thenReturn("user@jdw.com");

        ResponseEntity<Profile> response = profilesController.deleteIcon(profileId, "Bearer token");

        assertEquals(HttpStatus.NO_CONTENT, response.getStatusCode());
        verify(profileService).deleteIcon(profileId, "user@jdw.com");
    }
}