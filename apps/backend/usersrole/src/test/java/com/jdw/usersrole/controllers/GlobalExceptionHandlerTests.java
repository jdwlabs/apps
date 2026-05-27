package com.jdw.usersrole.controllers;

import com.jdw.usersrole.exceptions.IconUploadException;
import com.jdw.usersrole.exceptions.ResourceExistsException;
import com.jdw.usersrole.exceptions.ResourceNotFoundException;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.validation.BindingResult;
import org.springframework.validation.FieldError;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.multipart.MultipartException;
import org.springframework.web.multipart.support.MissingServletRequestPartException;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class GlobalExceptionHandlerTests {
    @InjectMocks
    GlobalExceptionHandler globalExceptionHandler;

    @Test
    void handleMethodArgumentNotValidException_shouldReturnBadRequest() {
        MethodArgumentNotValidException methodArgumentNotValidException = mock(MethodArgumentNotValidException.class);
        BindingResult bindingResult = mock(BindingResult.class);
        when(bindingResult.getAllErrors()).thenReturn(List.of(new FieldError("userRequestDTO", "emailAddress", "should not be empty")));
        when(methodArgumentNotValidException.getBindingResult()).thenReturn(bindingResult);

        ResponseEntity<Map<String, String>> response = globalExceptionHandler.handle(methodArgumentNotValidException);

        Map<String, String> expectedErrors = Map.of("emailAddress", "should not be empty");
        assertEquals(ResponseEntity.badRequest().body(expectedErrors), response);
    }

    @Test
    void handleResourceNotFoundException_shouldReturnNotFound() {
        ResourceNotFoundException ex = new ResourceNotFoundException("Resource not found");

        ResponseEntity<String> response = globalExceptionHandler.handle(ex);

        assertEquals(HttpStatus.NOT_FOUND, response.getStatusCode());
        assertEquals("Resource not found", response.getBody());
    }

    @Test
    void handleResourceExistsException_shouldReturnConflict() {
        ResourceExistsException ex = new ResourceExistsException("Resource already exists");

        ResponseEntity<String> response = globalExceptionHandler.handle(ex);

        assertEquals(HttpStatus.CONFLICT, response.getStatusCode());
        assertEquals("Resource already exists", response.getBody());
    }

    @Test
    void handleIconUploadException_shouldReturnInternalServerError() {
        IconUploadException ex = new IconUploadException("Icon upload failed", new Throwable());

        ResponseEntity<String> response = globalExceptionHandler.handle(ex);

        assertEquals(HttpStatus.INTERNAL_SERVER_ERROR, response.getStatusCode());
        assertEquals("Icon upload failed", response.getBody());
    }

    @Test
    void handleMultipartException_shouldReturnBadRequest() {
        MultipartException ex = new MultipartException("Multipart upload failed", new Throwable());

        ResponseEntity<String> response = globalExceptionHandler.handle(ex);

        assertEquals(HttpStatus.BAD_REQUEST, response.getStatusCode());
        assertEquals("Multipart upload failed", response.getBody());
    }

    @Test
    void handleHttpMessageNotReadableException_shouldReturnBadRequest() {
        HttpMessageNotReadableException ex = new HttpMessageNotReadableException("Http message is not readable", new Throwable());

        ResponseEntity<String> response = globalExceptionHandler.handle(ex);

        assertEquals(HttpStatus.BAD_REQUEST, response.getStatusCode());
        assertEquals("Request body is invalid. Please check the format and try again.", response.getBody());
    }

    @Test
    void handleMissingServletRequestPartException_shouldReturnBadRequest() {
        MissingServletRequestPartException ex = new MissingServletRequestPartException("Servlet request part is not supported");

        ResponseEntity<String> response = globalExceptionHandler.handle(ex);

        assertEquals(HttpStatus.BAD_REQUEST, response.getStatusCode());
        assertNotNull(response.getBody());
        assertTrue(response.getBody().contains("Servlet request part is not supported"));
    }

    @Test
    void handleUsernameNotFoundException_shouldReturnUnauthorized() {
        UsernameNotFoundException ex = new UsernameNotFoundException("Username not found");

        ResponseEntity<String> response = globalExceptionHandler.handle(ex);

        assertEquals(HttpStatus.UNAUTHORIZED, response.getStatusCode());
        assertEquals("Username not found", response.getBody());
    }
}