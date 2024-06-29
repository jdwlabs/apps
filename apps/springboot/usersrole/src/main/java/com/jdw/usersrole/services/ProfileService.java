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
import com.jdw.usersrole.repositories.ProfileRepository;
import com.jdw.usersrole.repositories.UserRepository;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotNull;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class ProfileService {
    private final ProfileRepository profileRepository;
    private final UserRepository userRepository;

    private static void logError(IOException e) {
        log.error("Could not update icon: error={}", e.toString());
    }

    public List<Profile> getProfiles() {
        log.info("Getting all profiles");
        return profileRepository.findAll();
    }

    public Profile getProfileById(@NotNull Long id) {
        log.info("Getting profile with id {}", id);
        return profileRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with id " + id));
    }

    public Profile getProfileByUserId(@NotNull Long id) {
        log.info("Getting profile with user id {}", id);
        return profileRepository.findByUserId(id)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with user id " + id));
    }

    public Profile createProfile(@Valid ProfileCreateRequestDTO profileDTO) {
        log.info("Creating profile {}", profileDTO);
        userRepository.findById(profileDTO.userId())
                .orElseThrow(() -> new ResourceNotFoundException("User not found with user id " + profileDTO.userId()));
        profileRepository.findByUserId(profileDTO.userId())
                .ifPresent(profile -> {
                    throw new ResourceExistsException("Profile already exists for user with id: " + profileDTO.userId());
                });
        Timestamp currentTime = Timestamp.from(Instant.now());
        Profile newProfile = Profile.builder()
                .firstName(profileDTO.firstName())
                .middleName(profileDTO.middleName())
                .lastName(profileDTO.lastName())
                .birthdate(profileDTO.birthdate())
                .userId(profileDTO.userId())
                .addresses(null)
                .icon(null)
                .createdByUserId(1L)
                .createdTime(currentTime)
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return profileRepository.save(newProfile);
    }

    public Profile updateProfileById(@NotNull Long id, @Valid ProfileUpdateRequestDTO profileDTO) {
        log.info("Updating profile with id {}", id);
        Profile currentProfile = profileRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with id " + id));
        Profile updatedProfile = buildUpdatedProfile(currentProfile, profileDTO, 1L);
        return profileRepository.save(updatedProfile);
    }

    public Profile updateProfileByUserId(@NotNull Long id, @Valid ProfileUpdateRequestDTO profileDTO) {
        log.info("Updating profile with user id {}", id);
        Profile currentProfile = profileRepository.findByUserId(id)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with user id " + id));
        Profile updatedProfile = buildUpdatedProfile(currentProfile, profileDTO, 1L);
        return profileRepository.save(updatedProfile);
    }

    public void deleteProfileById(@NotNull Long id) {
        log.info("Deleting profile with id {}", id);
        profileRepository.deleteById(id);
    }

    public void deleteProfileByUserId(@NotNull Long userId) {
        log.info("Deleting profile with user id {}", userId);
        profileRepository.deleteByUserId(userId);
    }

    public Profile addAddress(@NotNull Long id, @Valid AddressRequestDTO addressDTO) {
        log.info("Adding address with profile id {}", id);
        profileRepository.findByUserId(id)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with id " + id));
        Timestamp currentTime = Timestamp.from(Instant.now());
        Address newAddress = Address.builder()
                .id(null)
                .addressLine1(addressDTO.addressLine1())
                .addressLine2(addressDTO.addressLine2())
                .city(addressDTO.city())
                .stateProvince(addressDTO.stateProvince())
                .postalCode(addressDTO.postalCode())
                .country(addressDTO.country())
                .profileId(id)
                .createdByUserId(1L)
                .createdTime(currentTime)
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return profileRepository.saveAddress(newAddress);
    }

    public Profile updateAddress(@NotNull Long profileId, @NotNull Long addressId, @Valid AddressRequestDTO addressDTO) {
        log.info("Updating address with: profileId={}, addressId={}", profileId, addressId);
        Profile currentProfile = profileRepository.findByUserId(profileId)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with id " + profileId));
        Address currentAddress = currentProfile.addresses()
                .stream()
                .filter(ads -> ads.id().equals(addressId))
                .findFirst()
                .orElseThrow(() -> new ResourceNotFoundException("Address not found with id " + addressId));
        Timestamp currentTime = Timestamp.from(Instant.now());
        Address updatedAddress = Address.builder()
                .id(currentAddress.id())
                .addressLine1(addressDTO.addressLine1())
                .addressLine2(addressDTO.addressLine2())
                .city(addressDTO.city())
                .stateProvince(addressDTO.stateProvince())
                .postalCode(addressDTO.postalCode())
                .country(addressDTO.country())
                .profileId(currentAddress.profileId())
                .createdByUserId(currentProfile.createdByUserId())
                .createdTime(currentAddress.createdTime())
                .modifiedByUserId(1L)
                .modifiedTime(currentTime)
                .build();
        return profileRepository.saveAddress(updatedAddress);
    }

    public void deleteAddress(@NotNull Long profileId, @NotNull Long addressId) {
        log.info("Deleting address with: profileId={}, addressId={}", profileId, addressId);
        profileRepository.deleteAddressByAddressId(addressId);
    }

    public byte[] getIcon(@NotNull Long profileId) {
        log.info("Getting profile icon with profile id {}", profileId);
        Profile currentProfile = profileRepository.findById(profileId)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with id " + profileId));
        ProfileIcon currentIcon = currentProfile.icon();
        if (currentIcon == null) {
            throw new ResourceNotFoundException("Profile icon not found with id " + profileId);
        }
        return currentIcon.icon();
    }

    public Profile addIcon(@NotNull Long id, @NotNull MultipartFile file) {
        log.info("Adding icon with id {}", id);
        Profile currentProfile = profileRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with id " + id));
        ProfileIcon currentIcon = currentProfile.icon();
        if (currentIcon != null) {
            throw new ResourceExistsException("Icon already exists for profile with id: " + id);
        }
        try {
            byte[] icon = file.getBytes();
            Timestamp currentTime = Timestamp.from(Instant.now());
            ProfileIcon newProfileIcon = ProfileIcon.builder()
                    .id(null)
                    .profileId(id)
                    .icon(icon)
                    .createdByUserId(1L)
                    .createdTime(currentTime)
                    .modifiedByUserId(1L)
                    .modifiedTime(currentTime)
                    .build();
            return profileRepository.saveIcon(newProfileIcon);
        } catch (IOException e) {
            logError(e);
            throw new IconUploadException("Failed to add icon", e);
        }
    }

    public Profile updateIcon(@NotNull Long id, @NotNull MultipartFile file) {
        log.info("Updating icon with id {}", id);
        Profile currentProfile = profileRepository.findByUserId(id)
                .orElseThrow(() -> new ResourceNotFoundException("Profile not found with id " + id));
        try {
            byte[] icon = file.getBytes();
            ProfileIcon currentIcon = currentProfile.icon();
            Timestamp currentTime = Timestamp.from(Instant.now());
            ProfileIcon newProfileIcon = ProfileIcon.builder()
                    .id(currentIcon.id())
                    .profileId(currentIcon.profileId())
                    .icon(icon)
                    .createdByUserId(currentIcon.createdByUserId())
                    .createdTime(currentIcon.createdTime())
                    .modifiedByUserId(1L)
                    .modifiedTime(currentTime)
                    .build();
            return profileRepository.saveIcon(newProfileIcon);
        } catch (IOException e) {
            logError(e);
            throw new IconUploadException("Failed to update icon", e);
        }
    }

    public void deleteIcon(@NotNull Long id) {
        log.info("Deleting icon with id {}", id);
        profileRepository.deleteIconById(id);
    }

    private Profile buildUpdatedProfile(Profile currentProfile, ProfileUpdateRequestDTO profileDTO, Long userId) {
        Timestamp currentTime = Timestamp.from(Instant.now());
        return Profile.builder()
                .id(currentProfile.id())
                .firstName(profileDTO.firstName())
                .middleName(profileDTO.middleName())
                .lastName(profileDTO.lastName())
                .birthdate(profileDTO.birthdate())
                .userId(currentProfile.userId())
                .addresses(currentProfile.addresses())
                .icon(currentProfile.icon())
                .createdByUserId(currentProfile.createdByUserId())
                .createdTime(currentProfile.createdTime())
                .modifiedByUserId(userId)
                .modifiedTime(currentTime)
                .build();
    }
}
