package com.jdw.usersrole.services;

import com.jdw.usersrole.repositories.RoleRepository;
import com.jdw.usersrole.repositories.UserRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
@Slf4j
public class JwtUserDetailService implements UserDetailsService {
    private final UserRepository userRepository;
    private final RoleRepository roleRepository;
    @Override
    public UserDetails loadUserByUsername(String emailAddress) throws UsernameNotFoundException {
        log.info("Loading user details with: emailAddress={}", emailAddress);
        return null;
//        return userRepository.
//                .get(emailAddress)
//                .map(user -> new SecurityUser(user, roleRepository))
//                .orElseThrow(() -> new UsernameNotFoundException("Email address not found: emailAddress=" + emailAddress));
    }
}
