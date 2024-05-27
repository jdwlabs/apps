package com.jdw.usersrole.services;

import com.jdw.usersrole.models.User;
import com.jdw.usersrole.repositories.UserRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class UserService {
    private final UserRepository userRepository;

    public List<User> getAllUsers() {
        log.info("Getting all users");
        return userRepository.findAll();
    }

    public User getUserById(Long id) {
        log.info("Getting user by id: {}", id);
        return userRepository.findById(id).orElse(null);
    }

}
