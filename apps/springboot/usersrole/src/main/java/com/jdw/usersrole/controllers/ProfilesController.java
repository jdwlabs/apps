package com.jdw.usersrole.controllers;

import com.jdw.usersrole.dtos.AddressRequestDTO;
import com.jdw.usersrole.dtos.ProfileCreateRequestDTO;
import com.jdw.usersrole.dtos.ProfileUpdateRequestDTO;
import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.services.ProfileService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

import java.util.List;

@RestController
@RequiredArgsConstructor
@RequestMapping("/api/profiles")
@Slf4j
public class ProfilesController {
    private final ProfileService profileService;

    @PreAuthorize("hasAuthority('ADMIN')")
    @GetMapping
    public ResponseEntity<List<Profile>> getProfiles() {
        log.trace("Getting all profiles");
        return ResponseEntity.ok(profileService.getProfiles());
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @GetMapping("/{profileId}")
    public ResponseEntity<Profile> getProfileById(@PathVariable Long profileId) {
        log.trace("Getting profile with id {}", profileId);
        Profile profile = profileService.getProfileById(profileId);
        return ResponseEntity.ok(profile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()")
    @GetMapping("/user/{userId}")
    public ResponseEntity<Profile> getProfileByUserId(@PathVariable Long userId) {
        log.trace("Getting profile with user id {}", userId);
        Profile profile = profileService.getProfileByUserId(userId);
        return ResponseEntity.ok(profile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profile.userId() == authentication.principal.getUserId()")
    @PostMapping
    public ResponseEntity<Profile> createProfile(@Valid @RequestBody ProfileCreateRequestDTO profile) {
        log.trace("Creating profile {}", profile);
        Profile createdProfile = profileService.createProfile(profile);
        return ResponseEntity.status(HttpStatus.CREATED).body(createdProfile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @PutMapping("/{profileId}")
    public ResponseEntity<Profile> updateProfileById(@PathVariable Long profileId,
                                                     @Valid @RequestBody ProfileUpdateRequestDTO profile) {
        log.trace("Updating profile with id {}", profileId);
        Profile updatedProfile = profileService.updateProfileById(profileId, profile);
        return ResponseEntity.ok(updatedProfile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()")
    @PutMapping("/user/{userId}")
    public ResponseEntity<Profile> updateProfileByUserId(@PathVariable Long userId,
                                                         @Valid @RequestBody ProfileUpdateRequestDTO profile) {
        log.trace("Updating profile with user id {}", userId);
        Profile updatedProfile = profileService.updateProfileByUserId(userId, profile);
        return ResponseEntity.ok(updatedProfile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @DeleteMapping("/{profileId}")
    public ResponseEntity<Profile> deleteProfileById(@PathVariable Long profileId) {
        log.trace("Deleting profile with id {}", profileId);
        profileService.deleteProfileById(profileId);
        return ResponseEntity.noContent().build();
    }

    @PreAuthorize("hasAuthority('ADMIN') or #userId == authentication.principal.getUserId()")
    @DeleteMapping("/user/{userId}")
    public ResponseEntity<Profile> deleteProfileByUserId(@PathVariable Long userId) {
        log.trace("Deleting profile with user id {}", userId);
        profileService.deleteProfileByUserId(userId);
        return ResponseEntity.noContent().build();
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @PostMapping("/{profileId}/address")
    public ResponseEntity<Profile> addAddress(@PathVariable Long profileId,
                                              @Valid @RequestBody AddressRequestDTO address) {
        log.trace("Adding address with id {}", profileId);
        Profile updatedProfile = profileService.addAddress(profileId, address);
        return ResponseEntity.ok(updatedProfile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @PutMapping("/{profileId}/address/{addressId}")
    public ResponseEntity<Profile> updateAddress(@PathVariable Long profileId,
                                                 @PathVariable Long addressId,
                                                 @Valid @RequestBody AddressRequestDTO address) {
        log.trace("Updating address with id {}", addressId);
        Profile updatedProfile = profileService.updateAddress(profileId, addressId, address);
        return ResponseEntity.ok(updatedProfile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @DeleteMapping("/{profileId}/address/{addressId}")
    public ResponseEntity<Profile> deleteAddress(@PathVariable Long profileId, @PathVariable Long addressId) {
        log.trace("Deleting address with id {}", addressId);
        profileService.deleteAddress(profileId, addressId);
        return ResponseEntity.noContent().build();
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @GetMapping(value = "/{profileId}/icon", produces = MediaType.IMAGE_PNG_VALUE)
    public ResponseEntity<byte[]> getProfileIcon(@PathVariable Long profileId) {
        log.trace("Getting profile icon with id {}", profileId);
        return ResponseEntity.ok(profileService.getIcon(profileId));
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @PostMapping("/{profileId}/icon")
    public ResponseEntity<Profile> addIcon(@PathVariable Long profileId, @RequestParam("icon") MultipartFile file) {
        log.trace("Adding icon with id {}", profileId);
        Profile updatedProfile = profileService.addIcon(profileId, file);
        return ResponseEntity.ok(updatedProfile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @PutMapping("/{profileId}/icon")
    public ResponseEntity<Profile> updateIcon(@PathVariable Long profileId, @RequestParam("icon") MultipartFile file) {
        log.trace("Updating icon with id {}", profileId);
        Profile updatedProfile = profileService.updateIcon(profileId, file);
        return ResponseEntity.ok(updatedProfile);
    }

    @PreAuthorize("hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId()")
    @DeleteMapping("/{profileId}/icon")
    public ResponseEntity<Profile> deleteIcon(@PathVariable Long profileId) {
        log.trace("Deleting icon with id {}", profileId);
        profileService.deleteIcon(profileId);
        return ResponseEntity.noContent().build();
    }
}
