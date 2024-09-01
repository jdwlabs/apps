package com.jdw.usersrole.repositories;

import com.jdw.usersrole.daos.AddressDao;
import com.jdw.usersrole.daos.ProfileDao;
import com.jdw.usersrole.daos.ProfileIconDao;
import com.jdw.usersrole.models.Address;
import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.models.ProfileIcon;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.sql.Date;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.Collections;
import java.util.List;
import java.util.Optional;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class ProfileRepositoryImplTests {
    @Mock
    private ProfileDao profileDao;
    @Mock
    private ProfileIconDao profileIconDao;
    @Mock
    private AddressDao addressDao;
    @InjectMocks
    private ProfileRepositoryImpl profileRepository;

    private Profile buildMockProfile() {
        return Profile.builder()
                .id(1L)
                .firstName("John")
                .middleName("M")
                .lastName("Doe")
                .birthdate(Date.valueOf("1990-01-01"))
                .userId(1L)
                .addresses(Set.of())
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    private Address buildMockAddress() {
        return Address.builder()
                .id(1L)
                .profileId(1L)
                .addressLine1("123 Main St")
                .city("Cityville")
                .stateProvince("State")
                .country("Country")
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    private ProfileIcon buildMockProfileIcon() {
        return ProfileIcon.builder()
                .id(1L)
                .profileId(1L)
                .icon(new byte[]{1, 2, 3})
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();
    }

    @Test
    void findById_shouldReturnProfileWithAddressesAndIcon() {
        Profile mockProfile = buildMockProfile();
        Address mockAddress = buildMockAddress();
        ProfileIcon mockProfileIcon = buildMockProfileIcon();

        when(profileDao.findById(1L)).thenReturn(Optional.of(mockProfile));
        when(addressDao.findByProfileId(1L)).thenReturn(List.of(mockAddress));
        when(profileIconDao.findByProfileId(1L)).thenReturn(Optional.of(mockProfileIcon));

        Optional<Profile> result = profileRepository.findById(1L);

        assertTrue(result.isPresent());
        assertEquals(mockProfile.firstName(), result.get().firstName());
        assertEquals(1, result.get().addresses().size());
        assertEquals(mockProfileIcon, result.get().icon());
        verify(profileDao, times(1)).findById(1L);
        verify(addressDao, times(1)).findByProfileId(1L);
        verify(profileIconDao, times(1)).findByProfileId(1L);
    }

    @Test
    void findByUserId_shouldReturnProfileWithAddressesAndIcon() {
        Profile mockProfile = buildMockProfile();
        Address mockAddress = buildMockAddress();
        ProfileIcon mockProfileIcon = buildMockProfileIcon();

        when(profileDao.findByUserId(1L)).thenReturn(Optional.of(mockProfile));
        when(addressDao.findByProfileId(1L)).thenReturn(List.of(mockAddress));
        when(profileIconDao.findByProfileId(1L)).thenReturn(Optional.of(mockProfileIcon));

        Optional<Profile> result = profileRepository.findByUserId(1L);

        assertTrue(result.isPresent());
        assertEquals(mockProfile.firstName(), result.get().firstName());
        assertEquals(1, result.get().addresses().size());
        assertEquals(mockProfileIcon, result.get().icon());
        verify(profileDao, times(1)).findByUserId(1L);
        verify(addressDao, times(1)).findByProfileId(1L);
        verify(profileIconDao, times(1)).findByProfileId(1L);
    }

    @Test
    void findAll_shouldReturnAllProfilesWithAddressesAndIcons() {
        Profile mockProfile = buildMockProfile();
        Address mockAddress = buildMockAddress();
        ProfileIcon mockProfileIcon = buildMockProfileIcon();

        when(profileDao.findAll()).thenReturn(List.of(mockProfile));
        when(addressDao.findByProfileId(1L)).thenReturn(List.of(mockAddress));
        when(profileIconDao.findByProfileId(1L)).thenReturn(Optional.of(mockProfileIcon));

        List<Profile> result = profileRepository.findAll();

        assertEquals(1, result.size());
        assertEquals(mockProfile.firstName(), result.getFirst().firstName());
        assertEquals(1, result.getFirst().addresses().size());
        assertEquals(mockProfileIcon, result.getFirst().icon());
        verify(profileDao, times(1)).findAll();
        verify(addressDao, times(1)).findByProfileId(1L);
        verify(profileIconDao, times(1)).findByProfileId(1L);
    }

    @Test
    void save_shouldCreateNewProfileWhenIdIsNull() {
        Profile profileToSave = Profile.builder()
                .firstName("John")
                .middleName("M")
                .lastName("Doe")
                .birthdate(Date.valueOf("1990-01-01"))
                .userId(1L)
                .addresses(Set.of())
                .createdByUserId(1L)
                .createdTime(Timestamp.from(Instant.now()))
                .modifiedByUserId(1L)
                .modifiedTime(Timestamp.from(Instant.now()))
                .build();

        when(profileDao.create(profileToSave)).thenReturn(profileToSave);

        Profile result = profileRepository.save(profileToSave);

        assertEquals(profileToSave, result);
        verify(profileDao, times(1)).create(profileToSave);
        verify(profileDao, never()).update(any(Profile.class));
    }

    @Test
    void save_shouldUpdateProfileWhenIdIsPresent() {
        Profile mockProfile = buildMockProfile();

        when(profileDao.update(mockProfile)).thenReturn(mockProfile);
        when(addressDao.findByProfileId(1L)).thenReturn(Collections.emptyList());
        when(profileIconDao.findByProfileId(1L)).thenReturn(Optional.empty());

        Profile result = profileRepository.save(mockProfile);

        assertEquals(mockProfile, result);
        verify(profileDao, times(1)).update(mockProfile);
        verify(profileDao, never()).create(any(Profile.class));
    }

    @Test
    void save_shouldReturnNullWhenProfileIsNull() {
        Profile result = profileRepository.save(null);

        assertNull(result);
        verify(profileDao, never()).create(any());
        verify(profileDao, never()).update(any());
    }


    @Test
    void deleteById_shouldDeleteProfileAndRelatedData() {
        profileRepository.deleteById(1L);

        verify(addressDao, times(1)).deleteByProfileId(1L);
        verify(profileIconDao, times(1)).deleteByProfileId(1L);
        verify(profileDao, times(1)).deleteById(1L);
    }

    @Test
    void saveAddress_shouldSaveAddressAndReturnUpdatedProfile() {
        Address mockAddress = buildMockAddress();
        Profile mockProfile = buildMockProfile();

        when(addressDao.create(any(Address.class))).thenReturn(mockAddress);
        when(profileDao.findById(1L)).thenReturn(Optional.of(mockProfile));

        Profile updatedProfile = profileRepository.saveAddress(mockAddress);

        assertEquals(mockProfile, updatedProfile);
        verify(addressDao, times(1)).create(any(Address.class));
        verify(profileDao, times(1)).findById(1L);
    }

    @Test
    void saveAddress_ifNull_shouldReturnNull() {
        assertNull(profileRepository.saveAddress(null));
    }

    @Test
    void saveIcon_shouldSaveOrUpdateIconAndReturnUpdatedProfile() {
        ProfileIcon mockProfileIcon = buildMockProfileIcon();
        Profile mockProfile = buildMockProfile();

        when(profileIconDao.update(any(ProfileIcon.class))).thenReturn(mockProfileIcon);
        when(profileDao.findById(1L)).thenReturn(Optional.of(mockProfile));

        Profile updatedProfile = profileRepository.saveIcon(mockProfileIcon);

        assertEquals(mockProfile, updatedProfile);
        verify(profileIconDao, times(1)).update(any(ProfileIcon.class));
        verify(profileDao, times(1)).findById(1L);
    }

    @Test
    void deleteIconById_shouldDeleteIconByProfileId() {
        profileRepository.deleteIconById(1L);

        verify(profileIconDao, times(1)).deleteByProfileId(1L);
    }

    @Test
    void getProfile_shouldReturnEmptyOptionalWhenProfileIsNotPresent() {
        Optional<Profile> result = profileRepository.findById(1L);

        assertFalse(result.isPresent());
        verify(profileDao, times(1)).findById(1L);
        verify(addressDao, never()).findByProfileId(anyLong());
        verify(profileIconDao, never()).findByProfileId(anyLong());
    }

    @Test
    void deleteAddressByAddressId_shouldDeleteAddress() {
        profileRepository.deleteAddressByAddressId(1L);

        verify(addressDao, times(1)).deleteById(1L);
        verifyNoMoreInteractions(addressDao, profileDao, profileIconDao);
    }

    @Test
    void deleteByUserId_shouldDeleteProfileAndRelatedDataWhenProfileExists() {
        Profile mockProfile = buildMockProfile();
        when(profileDao.findByUserId(1L)).thenReturn(Optional.of(mockProfile));

        profileRepository.deleteByUserId(1L);

        verify(profileDao, times(1)).findByUserId(1L);
        verify(addressDao, times(1)).deleteByProfileId(1L);
        verify(profileIconDao, times(1)).deleteByProfileId(1L);
        verify(profileDao, times(1)).deleteByUserId(1L);
    }

    @Test
    void deleteByUserId_shouldNotDeleteRelatedDataWhenProfileDoesNotExist() {
        when(profileDao.findByUserId(1L)).thenReturn(Optional.empty());

        profileRepository.deleteByUserId(1L);

        verify(profileDao, times(1)).findByUserId(1L);
        verify(addressDao, never()).deleteByProfileId(anyLong());
        verify(profileIconDao, never()).deleteByProfileId(anyLong());
        verify(profileDao, times(1)).deleteByUserId(1L);
    }

    @Test
    void saveIcon_shouldReturnNullWhenIconIsNull() {
        Profile result = profileRepository.saveIcon(null);

        assertNull(result);
        verifyNoInteractions(profileIconDao, profileDao, addressDao);
    }

    @Test
    void saveIcon_shouldCreateNewIconWhenIconIdIsNull() {
        ProfileIcon mockProfileIcon = ProfileIcon.builder()
                .profileId(1L)
                .build();
        Profile mockProfile = buildMockProfile();

        when(profileDao.findById(1L)).thenReturn(Optional.of(mockProfile));

        Profile result = profileRepository.saveIcon(mockProfileIcon);

        verify(profileIconDao, times(1)).create(mockProfileIcon);
        verify(profileIconDao, never()).update(any(ProfileIcon.class));
        verify(profileDao, times(1)).findById(1L);
        assertEquals(mockProfile, result);
    }
}