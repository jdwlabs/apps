package com.jdw.usersrole.dtos;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.Builder;

@Builder
public record AuthResponseDTO(
        @NotNull(message = "jwtToken is mandatory")
        @NotBlank(message = "jwtToken is mandatory")
        String jwtToken
) {
}
