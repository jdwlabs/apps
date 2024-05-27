package com.jdw.usersrole.repositories;

import com.jdw.usersrole.models.Profile;
import org.springframework.data.repository.ListCrudRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface ProfileRepository extends ListCrudRepository<Profile, Long> {

}
