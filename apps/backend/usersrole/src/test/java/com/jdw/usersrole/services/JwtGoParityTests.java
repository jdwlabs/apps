package com.jdw.usersrole.services;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.jdw.usersrole.models.SecurityUser;
import io.jsonwebtoken.Claims;
import io.jsonwebtoken.io.Decoders;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.Arguments;
import org.junit.jupiter.params.provider.MethodSource;
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
import java.util.stream.Stream;

import static org.junit.jupiter.api.Assertions.assertArrayEquals;
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
 * <p>Two things are checked. First, that the production verification path
 * accepts a token the Go library minted — if it does not, the two
 * implementations disagree about the signature or the key derivation. Second,
 * that a token this service mints right now carries exactly the JOSE header,
 * claim names and JSON types the Go fixture carries — if it does not, they
 * disagree about the layout, which no signature check would catch.
 */
@ExtendWith(MockitoExtension.class)
@Tag("fast")
@Tag("unit")
class JwtGoParityTests {
    /**
     * Byte-identical to the secret the Go library's tests use, and to the one
     * JwtServiceTests injects. Both sides hard-code it so a fixture minted by
     * either verifies against the other without a shared environment.
     */
    private static final String PARITY_SECRET = "bXl0dGVzdHNlY3JldGtleWZvcmpzb253d2VidG9rZW4xMjM0NTY3ODkwIC1uCg==";
    /**
     * The same secret with its final padding removed, which is the shape the
     * deployed secret has: a length two more than a multiple of four. jjwt's
     * decoder sizes its output from the count of alphabet characters rather
     * than demanding a whole final quantum, so this service has always accepted
     * it — the Go library did not, and that asymmetry is what these tests pin.
     *
     * <p>Derived from the padded form rather than written out, so the two
     * cannot drift into being different keys and make the assertions vacuous.
     */
    private static final String PARITY_SECRET_UNPADDED = PARITY_SECRET.replace("=", "");
    private static final String ISSUER_ORIGIN = "http://localhost:8080";
    private static final String EXPIRATION_TIME_MS = "7200000";

    /**
     * Minted by the Go library's test minter. Its lifetime is deliberately far
     * longer than a real token: JwtService reads the wall clock and offers no
     * seam, so a realistic expiry would make this assertion time out rather
     * than test anything. Refresh it with the command in the library's README,
     * which prints a replacement to paste here.
     */
    private static final String GO_MINTED_TOKEN = "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.ko6vRk3iGxy_NsN2HQcZxr8CPhRll-8yFm85mh_gC7E"; // gitleaks:allow

    private static final String GO_MINTED_SUBJECT = "parity@jdw.com";
    private static final String GO_MINTED_TOKEN_ID = "6a2f1c34-9b7e-4d51-8f0a-2c6d5e4b3a19"; // gitleaks:allow
    private static final long GO_MINTED_USER_ID = 42L;
    private static final long GO_MINTED_PROFILE_ID = 7L;
    private static final long GO_MINTED_ISSUED_AT = 1735689600L;

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
    void extractAllClaims_shouldAcceptATokenTheGoLibraryMinted() {
        Claims claims = assertDoesNotThrow(() -> jwtService.extractAllClaims(GO_MINTED_TOKEN),
                "the Go library's token did not verify; the two implementations disagree about HS256 or the key");

        assertEquals(GO_MINTED_SUBJECT, claims.getSubject(), "sub");
        assertEquals(ISSUER_ORIGIN + "/auth/authenticate", claims.getIssuer(), "iss");
        assertEquals(Set.of(ISSUER_ORIGIN), claims.getAudience(), "aud");
        assertEquals(GO_MINTED_TOKEN_ID, claims.getId(), "jti");
        assertEquals(GO_MINTED_USER_ID, ((Number) claims.get("user_id")).longValue(), "user_id");
        assertEquals(GO_MINTED_PROFILE_ID, ((Number) claims.get("profile_id")).longValue(), "profile_id");
        assertEquals(List.of("ADMIN"), claims.get("roles", List.class), "roles");
        assertEquals(GO_MINTED_ISSUED_AT, claims.getNotBefore().toInstant().getEpochSecond(), "nbf");
        assertEquals(GO_MINTED_ISSUED_AT, claims.getIssuedAt().toInstant().getEpochSecond(), "iat");
    }

    @Test
    void extractEmailAddress_shouldReadTheSubjectOfAGoMintedToken() {
        String emailAddress = jwtService.extractEmailAddress(GO_MINTED_TOKEN);

        assertEquals(GO_MINTED_SUBJECT, emailAddress);
    }

    @Test
    void generateToken_shouldProduceTheSameClaimLayoutAsTheGoLibrary() throws Exception {
        stubPrincipal();
        String jvmToken = jwtService.generateToken(userDetails, ISSUER_ORIGIN);
        dumpJvmFixture(jvmToken);
        Claims jvmClaims = jwtService.extractAllClaims(jvmToken);
        Claims goClaims = jwtService.extractAllClaims(GO_MINTED_TOKEN);

        assertEquals(new TreeSet<>(goClaims.keySet()), new TreeSet<>(jvmClaims.keySet()),
                "the two implementations write different claim names");
        for (String name : jvmClaims.keySet()) {
            Object jvmValue = jvmClaims.get(name);
            Object goValue = goClaims.get(name);
            assertNotNull(goValue, name + " is null in the Go token but not in the JVM token");
            assertEquals(jvmValue.getClass(), goValue.getClass(),
                    name + " has a different JSON type on each side");
        }
        assertEquals(headerOf(jvmToken), headerOf(GO_MINTED_TOKEN),
                "the two implementations write different JOSE headers");
        assertEquals(ISSUER_ORIGIN + "/auth/authenticate", jvmClaims.getIssuer());
        assertEquals(Set.of(ISSUER_ORIGIN), jvmClaims.getAudience());
        assertEquals(Long.parseLong(EXPIRATION_TIME_MS),
                jvmClaims.getExpiration().getTime() - jvmClaims.getIssuedAt().getTime(),
                "the token lifetime moved off the configured expiration time");
    }

    @Test
    void decodersBase64_shouldReadAnUnpaddedSecretAsTheSameKeyAsThePaddedOne() {
        assertEquals(2, PARITY_SECRET_UNPADDED.length() % 4,
                "the unpadded fixture no longer has the deployed secret's shape");

        assertArrayEquals(Decoders.BASE64.decode(PARITY_SECRET), Decoders.BASE64.decode(PARITY_SECRET_UNPADDED),
                "jjwt read the two forms of one secret as different keys");
    }

    /**
     * The direction that matters for a cutover: the Go services mint under the
     * deployed secret, and this service has to accept what they mint. The token
     * is the one the Go side signed against the padded form, so verifying it
     * with the unpadded form injected proves both decoders arrive at the same
     * key bytes — nothing else makes an HMAC agree.
     */
    @Test
    void extractAllClaims_shouldAcceptAGoMintedTokenWhenTheSecretIsUnpadded() throws Exception {
        injectField(jwtService, "secretKey", PARITY_SECRET_UNPADDED);

        Claims claims = assertDoesNotThrow(() -> jwtService.extractAllClaims(GO_MINTED_TOKEN),
                "the Go library's token did not verify against the unpadded secret");

        assertEquals(GO_MINTED_SUBJECT, claims.getSubject(), "sub");
        assertEquals(GO_MINTED_TOKEN_ID, claims.getId(), "jti");
    }

    /**
     * And the other direction, across the padding boundary rather than across
     * the language one: a token this service mints with the unpadded secret
     * still verifies under the padded form, which is the state a rotation that
     * changed only the padding would leave two services in.
     */
    @Test
    void generateToken_shouldMintUnderAnUnpaddedSecretATokenThePaddedSecretVerifies() throws Exception {
        stubPrincipal();
        injectField(jwtService, "secretKey", PARITY_SECRET_UNPADDED);
        String token = jwtService.generateToken(userDetails, ISSUER_ORIGIN);

        injectField(jwtService, "secretKey", PARITY_SECRET);

        Claims claims = assertDoesNotThrow(() -> jwtService.extractAllClaims(token),
                "a token minted under the unpadded secret did not verify under the padded one");
        assertEquals(GO_MINTED_SUBJECT, claims.getSubject(), "sub");
    }

    /**
     * One key per HMAC variant jjwt can select, plus the deployed shape: 1534
     * bytes, encoded unpadded to 2046 characters. Each key comes from the same
     * arithmetic as the Go suite's parityBandKey, so neither side carries a key
     * literal and the two cannot drift apart. Each row carries the variant
     * Keys.hmacShaKeyFor selects for that length and the token the Go minter
     * produced under the same key; refresh the tokens with the command in the
     * library's README.
     */
    static Stream<Arguments> bands() {
        return Stream.of(
                Arguments.of("HS256", 32, "HS256",
                        "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.oURFk6r6xXfZpGWxhCn8f_QYaRmWi6sEBvlNB9tzpew"), // gitleaks:allow
                Arguments.of("HS384", 48, "HS384",
                        "eyJhbGciOiJIUzM4NCJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.F2LTul5M2rwV48OH0FDSb27NOIDKneKu80CWlPPi8KbBmwgckmHvUr1PVF5o9b9L"), // gitleaks:allow
                Arguments.of("HS512", 64, "HS512",
                        "eyJhbGciOiJIUzUxMiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.7arF5v8mQ33OOYZi_lLcc9-VMmSZkLddYm7VjX2BjHvrrow_cri8HCW7huKDRQRx7gHs_r4smwcmVSuyNIZoKg"), // gitleaks:allow
                Arguments.of("deployed", 1534, "HS512",
                        "eyJhbGciOiJIUzUxMiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.VVo34tIkRkzj8Iu5qAtQHizjv8EGQQU1djVmtZDu48uVApfn7o-DSxsiCTFtraEu40kRKHAvc9jIcmaez6TdvQ") // gitleaks:allow
        );
    }

    private static String bandSecret(int keyBytes) {
        byte[] key = new byte[keyBytes];
        for (int i = 0; i < keyBytes; i++) {
            key[i] = (byte) (i * 7 + keyBytes);
        }
        return Base64.getEncoder().withoutPadding().encodeToString(key);
    }

    /**
     * A verify alone would not prove the Go side chose the right variant: jjwt's
     * parser accepts any HMAC variant the header names so long as the key is long
     * enough for it. The header assertion is what pins the Go minter to the
     * variant this service would have signed with.
     */
    @ParameterizedTest(name = "{0} band, {1}-byte key")
    @MethodSource("bands")
    void extractAllClaims_shouldAcceptTheGoLibrarysTokenInEveryKeyLengthBand(
            String band, int keyBytes, String alg, String goToken) throws Exception {
        injectField(jwtService, "secretKey", bandSecret(keyBytes));

        assertEquals(alg, headerOf(goToken).get("alg").asText(),
                "the Go minter signed with a different variant than jjwt derives from this key");
        Claims claims = assertDoesNotThrow(() -> jwtService.extractAllClaims(goToken),
                "the Go library's " + band + " token did not verify");
        assertEquals(GO_MINTED_SUBJECT, claims.getSubject(), "sub");
        assertEquals(GO_MINTED_TOKEN_ID, claims.getId(), "jti");
    }

    /**
     * The variant this service signs with is the one the Go verifier will insist
     * on, so it has to be the one derived from the key length — which this
     * asserts against jjwt itself rather than against a copy of its thresholds.
     * The minted token is written out for the Go suite to verify.
     */
    @ParameterizedTest(name = "{0} band, {1}-byte key")
    @MethodSource("bands")
    void generateToken_shouldSignWithTheVariantTheKeyLengthSelects(
            String band, int keyBytes, String alg, String goToken) throws Exception {
        stubPrincipal();
        injectField(jwtService, "secretKey", bandSecret(keyBytes));

        String jvmToken = jwtService.generateToken(userDetails, ISSUER_ORIGIN);

        assertEquals(alg, headerOf(jvmToken).get("alg").asText(), "jjwt signed " + band + " with an unexpected variant");
        assertEquals(headerOf(goToken), headerOf(jvmToken), "the two implementations write different JOSE headers");
        dumpBandFixture(band, keyBytes, jvmToken);
    }

    private void dumpBandFixture(String band, int keyBytes, String token) throws Exception {
        Claims claims = jwtService.extractAllClaims(token);
        ObjectNode fixture = MAPPER.createObjectNode();
        fixture.put("band", band);
        fixture.put("keyBytes", keyBytes);
        fixture.put("token", token);
        fixture.put("iat", claims.getIssuedAt().toInstant().getEpochSecond());
        Path dump = JVM_FIXTURE_DUMP.resolveSibling("jvm-minted-band-" + band + ".json");
        Files.createDirectories(dump.getParent());
        Files.write(dump, MAPPER.writerWithDefaultPrettyPrinter().writeValueAsBytes(fixture));
    }

    /**
     * The user and profile ids have to be non-null for the layout comparison to
     * see the same JSON types the Go fixture carries; a null would be written
     * as JSON null and compare equal to nothing.
     */
    private void stubPrincipal() {
        when(userDetails.getUsername()).thenReturn(GO_MINTED_SUBJECT);
        when(userDetails.getUserId()).thenReturn(GO_MINTED_USER_ID);
        when(userDetails.getProfileId()).thenReturn(GO_MINTED_PROFILE_ID);
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

    private void dumpJvmFixture(String token) throws Exception {
        Claims claims = jwtService.extractAllClaims(token);
        ObjectNode fixture = MAPPER.createObjectNode();
        fixture.put("description", "Minted by JwtService.generateToken; paste into the Go library's parity test.");
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
        assertTrue(Files.exists(JVM_FIXTURE_DUMP));
    }

    private void injectField(Object targetObject, String fieldName, Object value) throws Exception {
        Field field = targetObject.getClass().getDeclaredField(fieldName);
        field.setAccessible(true);
        field.set(targetObject, value);
    }
}
