package com.jdw.usersrole.daos;

import com.jdw.usersrole.models.User;

import java.util.List;
import java.util.Optional;

public class UserDaoPostgres implements UserDao {
    @Override
    public User create(User user) {
        return null;
    }

    @Override
    public Optional<User> findById(Long id) {
        return Optional.empty();
    }

    @Override
    public List<User> findAll() {
        return List.of();
    }

    @Override
    public User update(User user) {
        return null;
    }

    @Override
    public void deleteById(Long id) {

    }
}
