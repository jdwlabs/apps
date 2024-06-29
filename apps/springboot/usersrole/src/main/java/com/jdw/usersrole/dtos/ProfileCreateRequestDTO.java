package com.jdw.usersrole.dtos;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Past;

import java.sql.Date;

public record ProfileCreateRequestDTO(
        @NotNull(message = "firstName is mandatory")
        @NotBlank(message = "firstName is mandatory")
        String firstName,
        String middleName,
        @NotNull(message = "lastName is mandatory")
        @NotBlank(message = "lastName is mandatory")
        String lastName,
        @NotNull(message = "birthdate is mandatory")
        @Past(message = "birthdate must be a past date")
        Date birthdate,
        @NotNull(message = "userId is mandatory")
        Long userId
) {
}
