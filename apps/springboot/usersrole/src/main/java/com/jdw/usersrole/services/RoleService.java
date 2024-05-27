package com.jdw.usersrole.services;

import com.jdw.usersrole.models.Role;
import com.jdw.usersrole.repositories.RoleRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class RoleService {
    private final RoleRepository roleRepository;

    public List<Role> getAllRoles() {
        log.info("Getting all roles");
        return roleRepository.findAll();
    }

    public Role getRoleById(Long id) {
        log.info("Getting role with id {}", id);
        return roleRepository.findById(id).orElse(null);
    }
}
