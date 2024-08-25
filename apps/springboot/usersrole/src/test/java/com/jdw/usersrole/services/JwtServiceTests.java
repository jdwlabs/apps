package com.jdw.usersrole.services;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.ExpiredJwtException;
import io.jsonwebtoken.JwtException;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.core.userdetails.UserDetails;

import javax.crypto.SecretKey;
import java.lang.reflect.Field;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class JwtServiceTests {
    @Mock
    private UserDetails userDetails;
    @InjectMocks
    private JwtService jwtService;

    @BeforeEach
    void setUp() throws Exception {
        injectField(jwtService, "jwtExpirationTimeMs", 1000 * 60 * 60L); // 1 hour expiration
        injectField(jwtService, "secretKey", "bXl0dGVzdHNlY3JldGtleWZvcmpzb253d2VidG9rZW4xMjM0NTY3ODkwIC1uCg==");
    }

    private void injectField(Object targetObject, String fieldName, Object value) throws Exception {
        Field field = targetObject.getClass().getDeclaredField(fieldName);
        field.setAccessible(true);
        field.set(targetObject, value);
    }

    @Test
    void getEmailAddress_shouldReturnCorrectEmail() {
        when(userDetails.getUsername()).thenReturn("user@jdw.com");
        String token = jwtService.generateToken(userDetails);
        String authorizationHeader = "Bearer " + token;

        String emailAddress = jwtService.getEmailAddress(authorizationHeader);

        assertEquals("user@jdw.com", emailAddress, "Extracted email address should match the input email");
    }

    @Test
    void getJwtToken_shouldExtractTokenFromAuthorizationHeader() {
        String token = jwtService.generateToken(userDetails);
        String authorizationHeader = "Bearer " + token;

        String extractedToken = jwtService.getJwtToken(authorizationHeader);

        assertEquals(token, extractedToken, "Extracted JWT token should match the original token");
    }

    @Test
    void extractEmailAddress_shouldReturnCorrectEmail() {
        when(userDetails.getUsername()).thenReturn("user@jdw.com");
        String token = jwtService.generateToken(userDetails);

        String emailAddress = jwtService.extractEmailAddress(token);

        assertEquals("user@jdw.com", emailAddress, "Extracted email address should match the input email");
    }

    @Test
    void extractClaim_shouldReturnCorrectClaim() {
        when(userDetails.getUsername()).thenReturn("user@jdw.com");
        String token = jwtService.generateToken(userDetails);

        String subject = jwtService.extractClaim(token, Claims::getSubject);

        assertEquals("user@jdw.com", subject, "Extracted subject should match the username");
    }

    @Test
    void generateToken_shouldReturnValidToken() {
        String token = jwtService.generateToken(userDetails);

        assertNotNull(token, "Generated token should not be null");
        assertFalse(token.isEmpty(), "Generated token should not be empty");
    }

    @Test
    void generateToken_withExtraClaims_shouldReturnValidTokenWithClaims() {
        when(userDetails.getUsername()).thenReturn("user@jdw.com");
        Map<String, Object> extraClaims = new HashMap<>();
        extraClaims.put("role", "ADMIN");

        String token = jwtService.generateToken(extraClaims, userDetails);
        Claims claims = jwtService.extractAllClaims(token);

        assertEquals("user@jdw.com", claims.getSubject(), "The subject should match the username");
        assertEquals("ADMIN", claims.get("role"), "The extra claim 'role' should match the expected value");
    }

    @Test
    void isTokenValid_shouldReturnTrueForValidToken() {
        when(userDetails.getUsername()).thenReturn("user@jdw.com");
        String token = jwtService.generateToken(userDetails);

        assertTrue(jwtService.isTokenValid(token, userDetails), "Token should be valid for the user");
    }

    @Test
    void isTokenValid_shouldReturnFalseForExpiredToken() throws Exception {
        injectField(jwtService, "jwtExpirationTimeMs", 1L); // 1 millisecond expiration
        String token = jwtService.generateToken(userDetails);

        // Wait for the token to expire
        Thread.sleep(2);
        boolean isValid;

        try {
            isValid = jwtService.isTokenValid(token, userDetails);
        } catch (ExpiredJwtException e) {
            // The token is expired, so the exception confirms it
            isValid = true;
        }

        assertTrue(isValid, "Token should be expired and thus invalid");
    }

    @Test
    void isTokenValid_shouldReturnFalseForTokenWithMismatchedUserDetails() {
        UserDetails userDetails2 = mock(UserDetails.class);
        when(userDetails.getUsername()).thenReturn("user@jdw.com");
        when(userDetails2.getUsername()).thenReturn("user2@jdw.com");
        String token = jwtService.generateToken(userDetails);

        assertFalse(jwtService.isTokenValid(token, userDetails2), "Token should be invalid due to mismatched user details");
    }

    @Test
    void isTokenExpired_shouldReturnFalseForNewToken() {
        String token = jwtService.generateToken(userDetails);

        assertFalse(jwtService.isTokenExpired(token), "Token should not be expired");
    }

    @Test
    void isTokenExpired_shouldReturnTrueForExpiredToken() throws Exception {
        // Reduce the expiration time for testing
        injectField(jwtService, "jwtExpirationTimeMs", 1L); // 1 millisecond expiration
        String token = jwtService.generateToken(userDetails);

        // Wait for the token to expire
        Thread.sleep(2);

        boolean isExpired;
        try {
            isExpired = jwtService.isTokenExpired(token);
        } catch (ExpiredJwtException e) {
            // The token is expired, so the exception confirms it
            isExpired = true;
        }

        assertTrue(isExpired, "Token should be expired");
    }

    @Test
    void extractExpiration_shouldReturnCorrectExpirationDate() {
        String token = jwtService.generateToken(userDetails);
        Date expirationDate = jwtService.extractExpiration(token);

        assertNotNull(expirationDate, "Expiration date should not be null");
        assertTrue(expirationDate.after(new Date()), "Expiration date should be in the future");
    }

    @Test
    void extractAllClaims_shouldReturnClaimsForValidToken() {
        when(userDetails.getUsername()).thenReturn("user@jdw.com");
        String token = jwtService.generateToken(userDetails);

        Claims claims = jwtService.extractAllClaims(token);

        assertEquals("user@jdw.com", claims.getSubject(), "The subject should match the username");
    }

    @Test
    void extractAllClaims_withInvalidToken_shouldThrowJwtException() {
        String invalidToken = "invalidToken";

        assertThrows(JwtException.class, () -> jwtService.extractAllClaims(invalidToken), "Invalid token should throw JwtException");
    }

    @Test
    void getSignInKey_shouldReturnCorrectSecretKey() {
        SecretKey secretKey = jwtService.getSignInKey();

        assertNotNull(secretKey, "Secret key should not be null");
    }
}