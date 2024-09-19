package com.jdw.usersrole.dtos;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;

public record UserRequestDTO(
        @NotNull(message = "emailAddress is mandatory")
        @NotBlank(message = "emailAddress is mandatory")
        @Email(message = "emailAddress is not valid", regexp = "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$")
        String emailAddress,
        @NotNull(message = "password is mandatory")
        @NotBlank(message = "password is mandatory")
        @Pattern(
                regexp = "^(?=.*[A-Z])(?=.*[a-z])(?=.*\\d)(?=.*[!@#$%^&*()_+\\-=\\[\\]{};':\"\\\\|,.<>\\/?~`])[A-Za-z\\d!@#$%^&*()_+\\-=\\[\\]{};':\"\\\\|,.<>\\/?~`]{8,}$",
                message = "Password must be at least 8 characters long, contain at least one uppercase letter, one lowercase letter, one number, and one special character. Special characters include: ! @ # $ % ^ & * ( ) _ + - = [ ] { } ; ' : \" \\ | , . < > / ? ~ `"
        )
        String password
) {
}
