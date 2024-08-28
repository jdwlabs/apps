package com.jdw.usersrole.services;

import com.jdw.usersrole.dtos.AddressRequestDTO;
import com.jdw.usersrole.dtos.ProfileCreateRequestDTO;
import com.jdw.usersrole.dtos.ProfileUpdateRequestDTO;
import com.jdw.usersrole.exceptions.IconUploadException;
import com.jdw.usersrole.exceptions.ResourceExistsException;
import com.jdw.usersrole.exceptions.ResourceNotFoundException;
import com.jdw.usersrole.models.Address;
import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.models.ProfileIcon;
import com.jdw.usersrole.models.User;
import com.jdw.usersrole.repositories.ProfileRepository;
import com.jdw.usersrole.repositories.UserRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.sql.Date;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;
import java.util.Optional;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class ProfileServiceTests {
    @Mock
    private ProfileRepository profileRepository;
    @Mock
    private UserRepository userRepository;
    @Mock
    private MultipartFile multipartFile;
    @InjectMocks
    private ProfileService profileService;

    private Profile profile;
    private ProfileCreateRequestDTO profileCreateRequestDTO;
    private ProfileUpdateRequestDTO profileUpdateRequestDTO;
    private AddressRequestDTO addressRequestDTO;
    private ProfileIcon profileIcon;
    private Address address;

    @BeforeEach
    void setUp() {
        profile = Profile.builder()
                .id(1L)
                .firstName("John")
                .middleName("M")
                .lastName("Doe")
                .birthdate(null)
                .userId(1L)
                .addresses(Set.of())
                .icon(null)
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();

        profileCreateRequestDTO = new ProfileCreateRequestDTO(
                 "John", "M", "Doe", Date.valueOf("1999-11-01") , 1L
        );

        profileUpdateRequestDTO = new ProfileUpdateRequestDTO(
                "John", "M", "Doe", null
        );

        addressRequestDTO = new AddressRequestDTO(
                "123 Street", "Suite 100", "New York", "NY", "10001", "USA"
        );

        profileIcon = ProfileIcon.builder()
                .id(1L)
                .profileId(1L)
                .icon(new byte[]{})
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();

        address = Address.builder()
                .id(1L)
                .addressLine1("123 Street")
                .addressLine2("Suite 100")
                .city("New York")
                .stateProvince("NY")
                .postalCode("10001")
                .country("USA")
                .profileId(1L)
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void getProfiles_shouldReturnAllProfiles() {
        when(profileRepository.findAll()).thenReturn(List.of(profile));

        List<Profile> profiles = profileService.getProfiles();

        assertEquals(1, profiles.size());
        verify(profileRepository, times(1)).findAll();
    }

    @Test
    void getProfileById_shouldReturnProfile_whenProfileExists() {
        when(profileRepository.findById(1L)).thenReturn(Optional.of(profile));

        Profile result = profileService.getProfileById(1L);

        assertEquals(profile, result);
        verify(profileRepository, times(1)).findById(1L);
    }

    @Test
    void getProfileById_shouldThrowResourceNotFoundException_whenProfileDoesNotExist() {
        when(profileRepository.findById(1L)).thenReturn(Optional.empty());

        assertThrows(ResourceNotFoundException.class, () -> profileService.getProfileById(1L));
        verify(profileRepository, times(1)).findById(1L);
    }

    @Test
    void getProfileByUserId_shouldReturnProfile_whenProfileExists() {
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.of(profile));

        Profile result = profileService.getProfileByUserId(1L);

        assertEquals(profile, result);
        verify(profileRepository, times(1)).findByUserId(1L);
    }

    @Test
    void getProfileByUserId_shouldThrowResourceNotFoundException_whenProfileDoesNotExist() {
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.empty());

        assertThrows(ResourceNotFoundException.class, () -> profileService.getProfileByUserId(1L));
        verify(profileRepository, times(1)).findByUserId(1L);
    }

    @Test
    void createProfile_shouldCreateProfile_whenUserExistsAndProfileDoesNotExist() {
        User user = mock(User.class);
        when(userRepository.findById(1L)).thenReturn(Optional.of(user));
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.empty());
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));
        when(user.id()).thenReturn(1L);
        when(profileRepository.save(any(Profile.class))).thenReturn(profile);

        Profile createdProfile = profileService.createProfile(profileCreateRequestDTO, "user@jdw.com");

        assertNotNull(createdProfile);
        verify(profileRepository, times(1)).save(any(Profile.class));
    }

    @Test
    void createProfile_shouldThrowResourceExistsException_whenProfileAlreadyExists() {
        when(userRepository.findById(1L)).thenReturn(Optional.of(mock(User.class)));
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.of(profile));

        assertThrows(ResourceExistsException.class, () -> profileService.createProfile(profileCreateRequestDTO, "user@jdw.com"));
    }

    @Test
    void updateProfileById_shouldUpdateProfile_whenProfileExists() {
        User user = mock(User.class);
        when(profileRepository.findById(1L)).thenReturn(Optional.of(profile));
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));
        when(user.id()).thenReturn(1L);
        when(profileRepository.save(any(Profile.class))).thenReturn(profile);

        Profile updatedProfile = profileService.updateProfileById(1L, profileUpdateRequestDTO, "user@jdw.com");

        assertNotNull(updatedProfile);
        verify(profileRepository, times(1)).save(any(Profile.class));
    }

    @Test
    void updateProfileByUserId_shouldUpdateProfile_whenProfileExists() {
        User user = mock(User.class);
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.of(profile));
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));
        when(user.id()).thenReturn(1L);
        when(profileRepository.save(any(Profile.class))).thenReturn(profile);

        Profile updatedProfile = profileService.updateProfileByUserId(1L, profileUpdateRequestDTO, "user@jdw.com");

        assertNotNull(updatedProfile);
        verify(profileRepository, times(1)).save(any(Profile.class));
    }

    @Test
    void deleteProfileById_shouldDeleteProfile_whenProfileExists() {
        doNothing().when(profileRepository).deleteById(1L);

        profileService.deleteProfileById(1L, "user@jdw.com");

        verify(profileRepository, times(1)).deleteById(1L);
    }

    @Test
    void deleteProfileByUserId_shouldDeleteProfile_whenProfileExists() {
        doNothing().when(profileRepository).deleteByUserId(1L);

        profileService.deleteProfileByUserId(1L, "user@jdw.com");

        verify(profileRepository, times(1)).deleteByUserId(1L);
    }

    @Test
    void addAddress_shouldAddAddress_whenProfileExists() {
        User user = mock(User.class);
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.of(profile));
        when(profileRepository.saveAddress(any(Address.class))).thenReturn(profile);
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));
        when(user.id()).thenReturn(1L);

        Profile updatedProfile = profileService.addAddress(1L, addressRequestDTO, "user@jdw.com");

        assertNotNull(updatedProfile);
        verify(profileRepository, times(1)).saveAddress(any(Address.class));
    }

    @Test
    void addAddress_shouldThrowResourceNotFoundException_whenProfileDoesNotExist() {
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.empty());

        assertThrows(ResourceNotFoundException.class, () -> profileService.addAddress(1L, addressRequestDTO, "user@jdw.com"));
    }

    @Test
    void updateAddress_shouldUpdateAddress_whenAddressExists() {
        profile = Profile.builder()
                .id(1L)
                .firstName("John")
                .addresses(Set.of(address))
                .build();
        User user = mock(User.class);

        when(profileRepository.findByUserId(1L)).thenReturn(Optional.of(profile));
        when(profileRepository.saveAddress(any(Address.class))).thenReturn(profile);
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));
        when(user.id()).thenReturn(1L);

        Profile updatedProfile = profileService.updateAddress(1L, 1L, addressRequestDTO, "user@jdw.com");

        assertNotNull(updatedProfile);
        verify(profileRepository, times(1)).saveAddress(any(Address.class));
    }

    @Test
    void updateAddress_shouldThrowResourceNotFoundException_whenProfileDoesNotExist() {
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.empty());

        assertThrows(ResourceNotFoundException.class, () -> profileService.updateAddress(1L, 1L, addressRequestDTO, "user@jdw.com"));
    }

    @Test
    void updateAddress_shouldThrowResourceNotFoundException_whenAddressDoesNotExist() {
        profile = Profile.builder()
                .id(1L)
                .firstName("John")
                .addresses(Set.of())
                .build();

        when(profileRepository.findByUserId(1L)).thenReturn(Optional.of(profile));

        assertThrows(ResourceNotFoundException.class, () -> profileService.updateAddress(1L, 1L, addressRequestDTO, "user@jdw.com"));
    }

    @Test
    void deleteAddress_shouldDeleteAddress_whenAddressExists() {
        doNothing().when(profileRepository).deleteAddressByAddressId(1L);

        profileService.deleteAddress(1L, 1L, "user@jdw.com");

        verify(profileRepository, times(1)).deleteAddressByAddressId(1L);
    }

    @Test
    void getIcon_shouldReturnIcon_whenProfileAndIconExist() {
        profile = Profile.builder()
                .id(1L)
                .icon(profileIcon)
                .build();

        when(profileRepository.findById(1L)).thenReturn(Optional.of(profile));

        byte[] iconData = profileService.getIcon(1L);

        assertNotNull(iconData);
        verify(profileRepository, times(1)).findById(1L);
    }

    @Test
    void getIcon_shouldThrowResourceNotFoundException_whenProfileDoesNotExist() {
        when(profileRepository.findById(1L)).thenReturn(Optional.empty());

        assertThrows(ResourceNotFoundException.class, () -> profileService.getIcon(1L));
    }

    @Test
    void getIcon_shouldThrowResourceNotFoundException_whenIconDoesNotExist() {
        profile = Profile.builder()
                .id(1L)
                .icon(null)
                .build();

        when(profileRepository.findById(1L)).thenReturn(Optional.of(profile));

        assertThrows(ResourceNotFoundException.class, () -> profileService.getIcon(1L));
    }

    @Test
    void addIcon_shouldAddIcon_whenIconDoesNotExist() throws IOException {
        User user = mock(User.class);
        when(profileRepository.findById(1L)).thenReturn(Optional.of(profile));
        when(multipartFile.getBytes()).thenReturn(new byte[]{});
        when(profileRepository.saveIcon(any(ProfileIcon.class))).thenReturn(profile);
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));
        when(user.id()).thenReturn(1L);

        Profile updatedProfile = profileService.addIcon(1L, multipartFile, "user@jdw.com");

        assertNotNull(updatedProfile);
        verify(profileRepository, times(1)).saveIcon(any(ProfileIcon.class));
    }

    @Test
    void addIcon_shouldThrowResourceExistsException_whenIconAlreadyExists() {
        profile = Profile.builder()
                .id(1L)
                .icon(profileIcon)
                .build();

        when(profileRepository.findById(1L)).thenReturn(Optional.of(profile));

        assertThrows(ResourceExistsException.class, () -> profileService.addIcon(1L, multipartFile, "user@jdw.com"));
    }

    @Test
    void addIcon_shouldThrowIconUploadException_whenIOExceptionOccurs() throws IOException {
        when(profileRepository.findById(1L)).thenReturn(Optional.of(profile));
        when(multipartFile.getBytes()).thenThrow(IOException.class);

        assertThrows(IconUploadException.class, () -> profileService.addIcon(1L, multipartFile, "user@jdw.com"));
    }

    @Test
    void updateIcon_shouldUpdateIcon_whenIconExists() throws IOException {
        profile = Profile.builder()
                .id(1L)
                .icon(profileIcon)
                .build();
        User user = mock(User.class);

        when(profileRepository.findByUserId(1L)).thenReturn(Optional.of(profile));
        when(multipartFile.getBytes()).thenReturn(new byte[]{});
        when(profileRepository.saveIcon(any(ProfileIcon.class))).thenReturn(profile);
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(user));
        when(user.id()).thenReturn(1L);

        Profile updatedProfile = profileService.updateIcon(1L, multipartFile, "user@jdw.com");

        assertNotNull(updatedProfile);
        verify(profileRepository, times(1)).saveIcon(any(ProfileIcon.class));
    }

    @Test
    void updateIcon_shouldThrowResourceNotFoundException_whenProfileDoesNotExist() {
        when(profileRepository.findByUserId(1L)).thenReturn(Optional.empty());

        assertThrows(ResourceNotFoundException.class, () -> profileService.updateIcon(1L, multipartFile, "user@jdw.com"));
    }

    @Test
    void updateIcon_shouldThrowIconUploadException_whenIOExceptionOccurs() throws IOException {
        profile = Profile.builder()
                .id(1L)
                .icon(profileIcon)
                .build();

        when(profileRepository.findByUserId(1L)).thenReturn(Optional.of(profile));
        when(multipartFile.getBytes()).thenThrow(IOException.class);

        assertThrows(IconUploadException.class, () -> profileService.updateIcon(1L, multipartFile, "user@jdw.com"));
    }

    @Test
    void deleteIcon_shouldDeleteIcon_whenProfileExists() {
        doNothing().when(profileRepository).deleteIconById(1L);

        profileService.deleteIcon(1L, "user@jdw.com");

        verify(profileRepository, times(1)).deleteIconById(1L);
    }

    @Test
    void getUserIdByEmailAddress_shouldReturnUserId_whenUserExists() {
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.of(mock(User.class)));

        profileService.getUserIdByEmailAddress("user@jdw.com");

        verify(userRepository, times(1)).findByEmailAddress("user@jdw.com");
    }

    @Test
    void getUserIdByEmailAddress_shouldThrowResourceNotFoundException_whenUserDoesNotExist() {
        when(userRepository.findByEmailAddress("user@jdw.com")).thenReturn(Optional.empty());

        assertThrows(ResourceNotFoundException.class, () -> profileService.getUserIdByEmailAddress("user@jdw.com"));
    }
}