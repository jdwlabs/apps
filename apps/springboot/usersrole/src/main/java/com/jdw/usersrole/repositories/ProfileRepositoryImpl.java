package com.jdw.usersrole.repositories;

import com.jdw.usersrole.daos.AddressDao;
import com.jdw.usersrole.daos.ProfileDao;
import com.jdw.usersrole.daos.ProfileIconDao;
import com.jdw.usersrole.models.Address;
import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.models.ProfileIcon;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

import java.util.HashSet;
import java.util.List;
import java.util.Optional;
import java.util.Set;

@Repository
@RequiredArgsConstructor
@Slf4j
public class ProfileRepositoryImpl implements ProfileRepository {
    private final ProfileDao profileDao;
    private final ProfileIconDao profileIconDao;
    private final AddressDao addressDao;

    @Override
    @Transactional(readOnly = true)
    public Optional<Profile> findById(Long id) {
        log.debug("Finding profile with id: {}", id);
        Optional<Profile> profile = profileDao.findById(id);
        return getProfile(profile);
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<Profile> findByUserId(Long id) {
        log.debug("Finding profile with user id: {}", id);
        Optional<Profile> profile = profileDao.findByUserId(id);
        return getProfile(profile);
    }

    @Override
    @Transactional(readOnly = true)
    public List<Profile> findAll() {
        log.debug("Finding all profiles");
        List<Profile> profiles = profileDao.findAll();
        return profiles.stream().map(this::getProfile).toList();
    }

    @Override
    @Transactional
    public Profile save(Profile profile) {
        log.debug("Saving profile: {}", profile);
        Profile updatedProfile;
        if (profile == null) {
            return null;
        } else if (profile.id() == null) {
            log.debug("Saving new profile: {}", profile);
            updatedProfile = profileDao.create(profile);
        } else {
            log.debug("Updating profile: {}", profile);
            updatedProfile = profileDao.update(profile);
        }
        return getProfile(updatedProfile);
    }

    @Override
    @Transactional
    public void deleteById(Long id) {
        log.debug("Deleting profile with id: {}", id);
        addressDao.deleteByProfileId(id);
        profileIconDao.deleteByProfileId(id);
        profileDao.deleteById(id);
    }

    @Override
    @Transactional
    public void deleteByUserId(Long id) {
        log.debug("Deleting profile with user id: {}", id);
        Profile profile = profileDao.findByUserId(id).orElse(null);
        if (profile != null) {
            addressDao.deleteByProfileId(profile.id());
            profileIconDao.deleteByProfileId(profile.id());
        }
        profileDao.deleteByUserId(id);
    }

    @Override
    @Transactional
    public Profile saveAddress(Address address) {
        if (address == null) {
            return null;
        }
        log.debug("Saving address with id: {}", address.profileId());
        addressDao.create(address);
        return getProfile(profileDao.findById(address.profileId())).orElse(null);
    }

    @Override
    @Transactional
    public void deleteAddressByAddressId(Long addressId) {
        log.debug("Deleting address with address id: {}", addressId);
        addressDao.deleteById(addressId);
    }

    @Override
    @Transactional
    public Profile saveIcon(ProfileIcon icon) {
        if (icon == null) {
            return null;
        } else if (icon.id() == null) {
            log.debug("Saving new icon: {}", icon);
            profileIconDao.create(icon);
        } else {
            log.debug("Updating icon: {}", icon);
            profileIconDao.update(icon);
        }
        return getProfile(profileDao.findById(icon.profileId())).orElse(null);
    }

    @Override
    @Transactional
    public void deleteIconById(Long id) {
        log.debug("Deleting icon with profile id: {}", id);
        profileIconDao.deleteByProfileId(id);
    }

    private Optional<Profile> getProfile(Optional<Profile> profile) {
        log.debug("Getting optional profile: {}", profile);
        if (profile.isPresent()) {
            Profile exisitingProfile = profile.get();
            List<Address> addresses = addressDao.findByProfileId(exisitingProfile.id());
            Set<Address> addressesSet = new HashSet<>(addresses);
            ProfileIcon icon = profileIconDao.findByProfileId(exisitingProfile.id()).orElse(null);
            Profile transformedProfile = Profile.builder()
                    .id(exisitingProfile.id())
                    .firstName(exisitingProfile.firstName())
                    .middleName(exisitingProfile.middleName())
                    .lastName(exisitingProfile.lastName())
                    .birthdate(exisitingProfile.birthdate())
                    .userId(exisitingProfile.userId())
                    .addresses(addressesSet)
                    .icon(icon)
                    .createdByUserId(exisitingProfile.createdByUserId())
                    .createdTime(exisitingProfile.createdTime())
                    .modifiedByUserId(exisitingProfile.modifiedByUserId())
                    .modifiedTime(exisitingProfile.modifiedTime())
                    .build();
            return Optional.ofNullable(transformedProfile);
        }
        return profile;
    }

    private Profile getProfile(Profile profile) {
        log.debug("Getting profile: {}", profile);
        List<Address> addresses = addressDao.findByProfileId(profile.id());
        Set<Address> addressesSet = new HashSet<>(addresses);
        ProfileIcon icon = profileIconDao.findByProfileId(profile.id()).orElse(null);
        return Profile.builder()
                .id(profile.id())
                .firstName(profile.firstName())
                .middleName(profile.middleName())
                .lastName(profile.lastName())
                .birthdate(profile.birthdate())
                .userId(profile.userId())
                .addresses(addressesSet)
                .icon(icon)
                .createdByUserId(profile.createdByUserId())
                .createdTime(profile.createdTime())
                .modifiedByUserId(profile.modifiedByUserId())
                .modifiedTime(profile.modifiedTime())
                .build();
    }
}
