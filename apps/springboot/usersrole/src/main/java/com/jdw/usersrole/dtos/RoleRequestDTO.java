package com.jdw.usersrole.dtos;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;

public record RoleRequestDTO(
        @NotNull(message = "name is mandatory")
        @NotBlank(message = "name is mandatory")
        String name,
        @NotNull(message = "description is mandatory")
        @NotBlank(message = "description is mandatory")
        String description
) {
}
