package com.jdw.usersrole.repositories;

import com.jdw.usersrole.models.User;

import java.util.List;
import java.util.Optional;

public interface UserRepository {
    Optional<User> findById(Long id);

    Optional<User> findByEmailAddress(String emailAddress);

    List<User> findAll();

    User save(User user);

    void deleteById(Long id);
}
