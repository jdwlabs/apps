package com.jdw.usersrole.services;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.jdw.usersrole.models.SecurityUser;
import io.jsonwebtoken.Claims;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.core.authority.SimpleGrantedAuthority;

import java.lang.reflect.Field;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Base64;
import java.util.List;
import java.util.Set;
import java.util.TreeSet;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.doReturn;
import static org.mockito.Mockito.when;

/**
 * The JVM half of the cross-implementation parity suite. Its Go counterpart
 * lives in libs/backend/shared/auth and asserts the mirror image: that a token
 * this class mints verifies there.
 *
 * <p>Two claims are checked here. First, that the production verification path
 * accepts a token the Go library minted — if it does not, the two
 * implementations disagree about the signature or the key derivation. Second,
 * that a token this service mints right now carries exactly the claim names and
 * JSON types the Go fixture carries — if it does not, they disagree about the
 * claim layout, which no signature check would catch.
 */
@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class JwtGoParityTests {
    /**
     * Byte-identical to the secret the Go library's tests use. Both sides
     * hard-code it so a fixture minted by either verifies against the other
     * without a shared environment.
     */
    private static final String PARITY_SECRET = "bXl0dGVzdHNlY3JldGtleWZvcmpzb253d2VidG9rZW4xMjM0NTY3ODkwIC1uCg==";
    private static final String ISSUER_ORIGIN = "http://localhost:8080";
    private static final String EXPIRATION_TIME_MS = "7200000";

    private static final Path GO_FIXTURE =
            Path.of("../../../libs/backend/shared/auth/testdata/go-minted-token.json");
    /**
     * Where a refreshed JVM fixture is written for the Go side to pick up. A
     * build artifact rather than a committed file, for the same reason the
     * served OpenAPI dump is one: it is evidence for a diff, not a source of
     * truth.
     */
    private static final Path JVM_FIXTURE_DUMP = Path.of("build/parity/jvm-minted-token.json");

    private static final ObjectMapper MAPPER = new ObjectMapper();

    @Mock
    private SecurityUser userDetails;
    @InjectMocks
    private JwtService jwtService;

    @BeforeEach
    void setUp() throws Exception {
        injectField(jwtService, "jwtExpirationTimeMs", EXPIRATION_TIME_MS);
        injectField(jwtService, "secretKey", PARITY_SECRET);
    }

    @Test
    void extractAllClaims_shouldAcceptATokenTheGoLibraryMinted() throws Exception {
        JsonNode fixture = readFixture();
        String token = fixture.get("token").asText();

        Claims claims = assertDoesNotThrow(() -> jwtService.extractAllClaims(token),
                "the Go library's token did not verify; the two implementations disagree about HS256 or the key");

        JsonNode expected = fixture.get("claims");
        assertEquals(expected.get("sub").asText(), claims.getSubject(), "sub");
        assertEquals(expected.get("iss").asText(), claims.getIssuer(), "iss");
        assertEquals(Set.of(expected.get("aud").asText()), claims.getAudience(), "aud");
        assertEquals(expected.get("jti").asText(), claims.getId(), "jti");
        assertEquals(expected.get("user_id").asLong(), ((Number) claims.get("user_id")).longValue(), "user_id");
        assertEquals(expected.get("profile_id").asLong(), ((Number) claims.get("profile_id")).longValue(), "profile_id");
        assertEquals(List.of("ADMIN"), claims.get("roles", List.class), "roles");
        assertEquals(expected.get("nbf").asLong(), claims.getNotBefore().toInstant().getEpochSecond(), "nbf");
        assertEquals(expected.get("iat").asLong(), claims.getIssuedAt().toInstant().getEpochSecond(), "iat");
    }

    @Test
    void extractEmailAddress_shouldReadTheSubjectOfAGoMintedToken() throws Exception {
        JsonNode fixture = readFixture();

        String emailAddress = jwtService.extractEmailAddress(fixture.get("token").asText());

        assertEquals(fixture.get("claims").get("sub").asText(), emailAddress);
    }

    @Test
    void generateToken_shouldProduceTheSameClaimLayoutAsTheGoLibrary() throws Exception {
        stubPrincipal();
        String jvmToken = jwtService.generateToken(userDetails, ISSUER_ORIGIN);
        dumpJvmFixture(jvmToken);
        Claims jvmClaims = jwtService.extractAllClaims(jvmToken);
        Claims goClaims = jwtService.extractAllClaims(readFixture().get("token").asText());

        assertEquals(new TreeSet<>(goClaims.keySet()), new TreeSet<>(jvmClaims.keySet()),
                "the two implementations write different claim names");
        for (String name : jvmClaims.keySet()) {
            Object jvmValue = jvmClaims.get(name);
            Object goValue = goClaims.get(name);
            assertNotNull(goValue, name + " is null in the Go token but not in the JVM token");
            assertEquals(jvmValue.getClass(), goValue.getClass(),
                    name + " has a different JSON type on each side");
        }
        assertEquals(headerOf(jvmToken), headerOf(readFixture().get("token").asText()),
                "the two implementations write different JOSE headers");
        assertEquals(ISSUER_ORIGIN + "/auth/authenticate", jvmClaims.getIssuer());
        assertEquals(Set.of(ISSUER_ORIGIN), jvmClaims.getAudience());
        assertTrue(jvmClaims.getExpiration().getTime() - jvmClaims.getIssuedAt().getTime()
                        == Long.parseLong(EXPIRATION_TIME_MS),
                "the token lifetime moved off the configured expiration time");
    }

    /**
     * The user and profile ids have to be non-null for the layout comparison to
     * see the same JSON types the Go fixture carries; a null would be written
     * as JSON null and compare equal to nothing.
     */
    private void stubPrincipal() {
        when(userDetails.getUsername()).thenReturn("parity@jdw.com");
        when(userDetails.getUserId()).thenReturn(42L);
        when(userDetails.getProfileId()).thenReturn(7L);
        doReturn(List.of(new SimpleGrantedAuthority("ADMIN"))).when(userDetails).getAuthorities();
    }

    /**
     * The header is compared as parsed JSON rather than as text because base64
     * of the same object is not unique across encoders.
     */
    private JsonNode headerOf(String token) throws Exception {
        String segment = token.substring(0, token.indexOf('.'));
        return MAPPER.readTree(Base64.getUrlDecoder().decode(segment));
    }

    private JsonNode readFixture() throws Exception {
        assertTrue(Files.exists(GO_FIXTURE),
                "missing Go-minted fixture at " + GO_FIXTURE.toAbsolutePath().normalize());
        return MAPPER.readTree(Files.readAllBytes(GO_FIXTURE));
    }

    private void dumpJvmFixture(String token) throws Exception {
        Claims claims = jwtService.extractAllClaims(token);
        ObjectNode fixture = MAPPER.createObjectNode();
        fixture.put("description", "Minted by JwtService.generateToken; refresh with the parity test that writes it.");
        fixture.put("secretKeyBase64", PARITY_SECRET);
        fixture.put("issuerOrigin", ISSUER_ORIGIN);
        fixture.put("token", token);
        ObjectNode claimsNode = fixture.putObject("claims");
        claimsNode.put("sub", claims.getSubject());
        claimsNode.putArray("roles").add("ADMIN");
        claimsNode.put("user_id", ((Number) claims.get("user_id")).longValue());
        claimsNode.put("profile_id", ((Number) claims.get("profile_id")).longValue());
        claimsNode.put("aud", claims.getAudience().iterator().next());
        claimsNode.put("iss", claims.getIssuer());
        claimsNode.put("jti", claims.getId());
        claimsNode.put("iat", claims.getIssuedAt().toInstant().getEpochSecond());
        claimsNode.put("nbf", claims.getNotBefore().toInstant().getEpochSecond());
        claimsNode.put("exp", claims.getExpiration().toInstant().getEpochSecond());
        Files.createDirectories(JVM_FIXTURE_DUMP.getParent());
        Files.write(JVM_FIXTURE_DUMP, MAPPER.writerWithDefaultPrettyPrinter().writeValueAsBytes(fixture));
    }

    private void injectField(Object targetObject, String fieldName, Object value) throws Exception {
        Field field = targetObject.getClass().getDeclaredField(fieldName);
        field.setAccessible(true);
        field.set(targetObject, value);
    }
}
