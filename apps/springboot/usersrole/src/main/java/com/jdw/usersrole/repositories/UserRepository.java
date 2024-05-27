package com.jdw.usersrole.repositories;

import com.jdw.usersrole.models.User;
import org.springframework.data.repository.ListCrudRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface UserRepository extends ListCrudRepository<User, Long> {

}
