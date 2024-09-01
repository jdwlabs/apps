package com.jdw.usersrole.repositories;

import com.jdw.usersrole.daos.*;
import com.jdw.usersrole.models.*;
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
public class UserRepositoryImpl implements UserRepository {
    private final UserDao userDao;
    private final UserRoleDao userRoleDao;
    private final ProfileDao profileDao;
    private final ProfileIconDao profileIconDao;
    private final AddressDao addressDao;

    @Override
    @Transactional(readOnly = true)
    public Optional<User> findById(Long id) {
        log.debug("Finding user with id: {}", id);
        Optional<User> user = userDao.findById(id);
        return getUser(user);
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<User> findByEmailAddress(String emailAddress) {
        log.debug("Finding user with email address: {}", emailAddress);
        Optional<User> user = userDao.findByEmailAddress(emailAddress);
        return getUser(user);
    }

    @Override
    @Transactional(readOnly = true)
    public List<User> findAll() {
        log.debug("Finding all users");
        List<User> users = userDao.findAll();
        return users.stream().map(this::getUser).toList();
    }

    @Override
    @Transactional
    public User save(User user) {
        log.debug("Saving user: {}", user);
        User updatedUser;
        if (user == null) {
            return null;
        } else if (user.id() == null) {
            log.debug("Saving new user: {}", user);
            updatedUser = userDao.create(user);
        } else {
            log.debug("Updating user: {}", user);
            updatedUser = userDao.update(user);
        }
        return getUser(updatedUser);
    }

    @Override
    @Transactional
    public void deleteById(Long id) {
        log.debug("Deleting user with id: {}", id);
        Profile profile = getProfile(id);
        if (profile != null) {
            addressDao.deleteByProfileId(profile.id());
            profileIconDao.deleteByProfileId(profile.id());
        }
        profileDao.deleteByUserId(id);
        userRoleDao.deleteByUserId(id);
        userDao.deleteById(id);
    }

    private Optional<User> getUser(Optional<User> user) {
        log.debug("Getting optional user: {}", user);
        if (user.isPresent()) {
            User exisitingUser = user.get();
            List<UserRole> userRoles = userRoleDao.findByUserId(exisitingUser.id());
            Set<UserRole> userRolesSet = new HashSet<>(userRoles);
            Profile profile = getProfile(exisitingUser.id());
            User transformedUser = User.builder()
                    .id(exisitingUser.id())
                    .emailAddress(exisitingUser.emailAddress())
                    .password(exisitingUser.password())
                    .status(exisitingUser.status())
                    .roles(userRolesSet)
                    .profile(profile)
                    .createdByUserId(exisitingUser.createdByUserId())
                    .createdTime(exisitingUser.createdTime())
                    .modifiedByUserId(exisitingUser.modifiedByUserId())
                    .modifiedTime(exisitingUser.modifiedTime())
                    .build();
            return Optional.ofNullable(transformedUser);
        }
        return user;
    }

    private User getUser(User user) {
        log.debug("Getting user: {}", user);
        List<UserRole> userRoles = userRoleDao.findByUserId(user.id());
        Set<UserRole> userRolesSet = new HashSet<>(userRoles);
        Profile profile = getProfile(user.id());
        return User.builder()
                .id(user.id())
                .emailAddress(user.emailAddress())
                .password(user.password())
                .status(user.status())
                .roles(userRolesSet)
                .profile(profile)
                .createdByUserId(user.createdByUserId())
                .createdTime(user.createdTime())
                .modifiedByUserId(user.modifiedByUserId())
                .modifiedTime(user.modifiedTime())
                .build();
    }

    private Profile getProfile(Long id) {
        log.debug("Getting profile: {}", id);
        Optional<Profile> profileOptional = profileDao.findByUserId(id);
        if (profileOptional.isPresent()) {
            Profile profile = profileOptional.get();
            List<Address> addresses = addressDao.findByProfileId(profile.id());
            Set<Address> addressesSet = new HashSet<>(addresses);
            ProfileIcon icon = profileIconDao.findByProfileId(profile.id()).orElse(null);
            return Profile.builder()
                    .id(profile.id())
                    .firstName(profile.firstName())
                    .middleName(profile.middleName())
                    .lastName(profile.lastName())
                    .birthdate(profile.birthdate())
                    .userId(id)
                    .addresses(addressesSet)
                    .icon(icon)
                    .createdByUserId(profile.createdByUserId())
                    .createdTime(profile.createdTime())
                    .modifiedByUserId(profile.modifiedByUserId())
                    .modifiedTime(profile.modifiedTime())
                    .build();
        }
        return null;
    }
}
