package com.jdw.usersrole.services;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.JwtException;
import io.jsonwebtoken.Jwts;
import io.jsonwebtoken.io.Decoders;
import io.jsonwebtoken.security.Keys;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.stereotype.Service;

import javax.crypto.SecretKey;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.function.Function;

@Service
@Slf4j
public class JwtService {
    @Value("${app.jwt.expiration-time-ms}")
    private long jwtExpirationTimeMs;
    @Value("${app.jwt.secret-key}")
    private String secretKey;

    public String getEmailAddress(String authorizationHeader) {
        log.info("Retrieving email address with: authorizationHeader={}", authorizationHeader);
        return extractEmailAddress(getJwtToken(authorizationHeader));
    }

    public String getJwtToken(String authorizationHeader) {
        log.info("Retrieving JWT token with: authorizationHeader={}", authorizationHeader);
        return authorizationHeader.substring(7);
    }

    public String extractEmailAddress(String jwtToken) {
        log.info("Extracting email address with: jwtToken={}", jwtToken);
        return extractClaim(jwtToken, Claims::getSubject);
    }

    public <T> T extractClaim(String jwtToken, Function<Claims, T> claimsResolver) {
        log.info("Extracting claim with: jwtToken={}", jwtToken);
        return claimsResolver.apply(extractAllClaims(jwtToken));
    }

    public String generateToken(UserDetails userDetails) {
        log.info("Generating empty extra claims token with: emailAddress={}", userDetails.getUsername());
        return generateToken(new HashMap<>(), userDetails);
    }

    public String generateToken(Map<String, Object> extraClaims, UserDetails userDetails) {
        log.info("Generating token with: emailAddress={}", userDetails.getUsername());
        return Jwts
                .builder()
                .claims(extraClaims)
                .subject(userDetails.getUsername())
                .issuedAt(new Date())
                .expiration(new Date(System.currentTimeMillis() + jwtExpirationTimeMs))
                .signWith(getSignInKey())
                .compact();
    }

    public boolean isTokenValid(String jwtToken, UserDetails userDetails) {
        log.info("Checking token validation with: jwtToken={}", jwtToken);
        return extractEmailAddress(jwtToken).equals(userDetails.getUsername()) && !isTokenExpired(jwtToken);
    }

    public boolean isTokenExpired(String jwtToken) {
        log.info("Checking token expiration with: jwtToken={}", jwtToken);
        return extractExpiration(jwtToken).before(new Date());
    }

    protected Date extractExpiration(String jwtToken) {
        log.info("Extracting expiration with: jwtToken={}", jwtToken);
        return extractClaim(jwtToken, Claims::getExpiration);
    }

    protected Claims extractAllClaims(String jwtToken) throws JwtException {
        log.info("Extracting all claims with: jwtToken={}", jwtToken);
        return Jwts
                .parser()
                .verifyWith(getSignInKey())
                .build()
                .parseSignedClaims(jwtToken)
                .getPayload();
    }

    protected SecretKey getSignInKey() {
        log.info("Retrieving sign in key");
        byte[] keyBytes = Decoders.BASE64.decode(secretKey);
        return Keys.hmacShaKeyFor(keyBytes);
    }
}
