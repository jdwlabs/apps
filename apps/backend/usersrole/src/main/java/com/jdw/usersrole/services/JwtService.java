package com.jdw.usersrole.services;

import com.jdw.usersrole.metrics.ExecutionTimeLogger;
import com.jdw.usersrole.models.SecurityUser;
import io.jsonwebtoken.Claims;
import io.jsonwebtoken.JwtException;
import io.jsonwebtoken.Jwts;
import io.jsonwebtoken.io.Decoders;
import io.jsonwebtoken.security.Keys;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.stereotype.Service;

import javax.crypto.SecretKey;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;
import java.util.function.Function;
import java.util.stream.Collectors;

@Service
@Slf4j
public class JwtService {
    @Value("${app.jwt.expiration-time-ms}")
    private String jwtExpirationTimeMs;
    @Value("${app.jwt.secret-key}")
    private String secretKey;

    public String getEmailAddress(String authorizationHeader) {
        log.debug("Retrieving email address with: authorizationHeader={}", authorizationHeader);
        return extractEmailAddress(getJwtToken(authorizationHeader));
    }

    public String getJwtToken(String authorizationHeader) {
        log.debug("Retrieving JWT token with: authorizationHeader={}", authorizationHeader);
        return authorizationHeader.substring(7);
    }

    public String extractEmailAddress(String jwtToken) {
        log.debug("Extracting email address with: jwtToken={}", jwtToken);
        return extractClaim(jwtToken, Claims::getSubject);
    }

    public <T> T extractClaim(String jwtToken, Function<Claims, T> claimsResolver) {
        log.debug("Extracting claim with: jwtToken={}", jwtToken);
        return claimsResolver.apply(extractAllClaims(jwtToken));
    }

    public String generateToken(UserDetails userDetails, String issuerUrl) {
        log.info("Generating empty extra claims token with: emailAddress={}", userDetails.getUsername());
        return generateToken(new HashMap<>(), userDetails, issuerUrl);
    }

    @ExecutionTimeLogger
    public String generateToken(Map<String, Object> extraClaims, UserDetails userDetails, String issuerUrl) {
        log.info("Generating token with: emailAddress={}", userDetails.getUsername());
        Date now = new Date();
        Date expiration = new Date(System.currentTimeMillis() + getJwtExpirationTimeMs());
        Map<String, Object> claims = new HashMap<>(extraClaims);
        claims.put("roles", userDetails.getAuthorities().stream()
                .map(GrantedAuthority::getAuthority)
                .collect(Collectors.toList()));
        claims.put("user_id", ((SecurityUser) userDetails).getUserId());
        claims.put("profile_id", ((SecurityUser) userDetails).getProfileId());
        claims.put("aud", issuerUrl);
        claims.put("iss", issuerUrl + "/auth/authenticate");
        claims.put("jti", UUID.randomUUID().toString());
        claims.put("nbf", now);
        return Jwts
                .builder()
                .claims(claims)
                .subject(userDetails.getUsername())
                .issuedAt(now)
                .expiration(expiration)
                .signWith(getSignInKey())
                .compact();
    }

    public boolean isTokenValid(String jwtToken, UserDetails userDetails) {
        log.debug("Checking token validation with: jwtToken={}", jwtToken);
        return extractEmailAddress(jwtToken).equals(userDetails.getUsername()) && !isTokenExpired(jwtToken);
    }

    public boolean isTokenExpired(String jwtToken) {
        log.debug("Checking token expiration with: jwtToken={}", jwtToken);
        return extractExpiration(jwtToken).before(new Date());
    }

    protected Date extractExpiration(String jwtToken) {
        log.debug("Extracting expiration with: jwtToken={}", jwtToken);
        return extractClaim(jwtToken, Claims::getExpiration);
    }

    protected Claims extractAllClaims(String jwtToken) throws JwtException {
        log.debug("Extracting all claims with: jwtToken={}", jwtToken);
        return Jwts
                .parser()
                .verifyWith(getSignInKey())
                .build()
                .parseSignedClaims(jwtToken)
                .getPayload();
    }

    protected SecretKey getSignInKey() {
        log.debug("Retrieving sign in key");
        byte[] keyBytes = Decoders.BASE64.decode(secretKey);
        return Keys.hmacShaKeyFor(keyBytes);
    }

    private long getJwtExpirationTimeMs() {
        try {
            return Long.parseLong(jwtExpirationTimeMs);
        } catch (NumberFormatException e) {
            log.error("Invalid JWT expiration time format: {}, error={}", jwtExpirationTimeMs, e.toString());
            throw e;
        }
    }
}
