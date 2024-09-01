package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.User;

import java.util.List;
import java.util.Optional;

public interface UserDao {
    User create(User user);

    Optional<User> findById(Long id);

    Optional<User> findByEmailAddress(String emailAddress);

    List<User> findAll();

    User update(User user);

    void deleteById(Long id);
}
