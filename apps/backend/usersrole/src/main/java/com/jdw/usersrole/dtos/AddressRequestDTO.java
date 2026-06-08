package com.jdw.usersrole.dtos;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;

public record AddressRequestDTO(
        @NotNull(message = "addressLine1 is mandatory")
        @NotBlank(message = "addressLine1 is mandatory")
        String addressLine1,
        String addressLine2,
        @NotNull(message = "city is mandatory")
        @NotBlank(message = "city is mandatory")
        String city,
        @NotNull(message = "stateProvince is mandatory")
        @NotBlank(message = "stateProvince is mandatory")
        String stateProvince,
        @NotNull(message = "postalCode is mandatory")
        @NotBlank(message = "postalCode is mandatory")
        String postalCode,
        @NotNull(message = "country is mandatory")
        @NotBlank(message = "country is mandatory")
        String country
) {
}
