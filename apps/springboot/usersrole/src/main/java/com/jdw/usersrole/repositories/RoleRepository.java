package com.jdw.usersrole.repositories;

import com.jdw.usersrole.models.Role;
import org.springframework.data.repository.ListCrudRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface RoleRepository extends ListCrudRepository<Role, Long> {

}
