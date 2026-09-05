package com.jdw.usersrole.controllers;

import com.jdw.usersrole.models.Profile;
import com.jdw.usersrole.services.ProfileService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.MediaType;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.security.test.context.support.WithMockUser;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;
import org.springframework.web.method.HandlerMethod;
import org.springframework.web.HttpMediaTypeNotAcceptableException;
import org.springframework.web.servlet.HandlerExecutionChain;
import org.springframework.web.servlet.mvc.method.annotation.RequestMappingHandlerMapping;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;
import org.springframework.web.util.ServletRequestPathUtils;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.multipart;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

/**
 * Asserts which handler the running {@link RequestMappingHandlerMapping} picks
 * for the URIs where the profile routes overlap, and what a client gets back.
 *
 * The overlap is real: {@code {profileId}} captures any literal, so a by-user
 * path sitting in the same position as a profile id competes with every literal
 * child of {@code /api/profiles/{profileId}}. It cannot be reasoned about from
 * the pattern comparator alone either, because two things happen in order — a
 * mapping whose method or {@code produces} condition does not match the request
 * is dropped first, and only the survivors are compared, pattern before
 * everything else. So {@code produces} decides a case only where the patterns
 * tie, which is exactly what used to happen here.
 */
@Tag("fast")
@Tag("integration")
@SpringBootTest
class ProfileRouteResolutionTests {
    private static final String PREVIOUSLY_AMBIGUOUS_URI = "/api/profiles/user/icon";
    private static final String ICON_LITERAL_UNDER_BY_USER = "/api/profiles/by-user/icon";
    private static final String ADDRESS_LITERAL_UNDER_BY_USER = "/api/profiles/by-user/address";
    private static final long USER_ID = 7L;

    @Autowired
    @Qualifier("requestMappingHandlerMapping")
    private RequestMappingHandlerMapping handlerMapping;

    @Autowired
    private WebApplicationContext context;

    @MockitoBean
    private ProfileService profileService;

    private MockMvc mockMvc;

    @BeforeEach
    void buildMockMvc() {
        // Filters are left out on purpose: the ambiguity under test is resolved by
        // the DispatcherServlet, which an unauthenticated request never reaches.
        mockMvc = MockMvcBuilders.webAppContextSetup(context).build();
    }

    private String resolve(String httpMethod, String uri, String accept) throws Exception {
        MockHttpServletRequest request = new MockHttpServletRequest(httpMethod, uri);
        if (accept != null) {
            request.addHeader(HttpHeaders.ACCEPT, accept);
        }
        ServletRequestPathUtils.parseAndCache(request);
        HandlerExecutionChain chain = handlerMapping.getHandler(request);
        assertNotNull(chain, httpMethod + " " + uri + " resolves to no handler at all");
        return ((HandlerMethod) chain.getHandler()).getMethod().getName();
    }

    @Test
    void getOnThePreviouslyAmbiguousUriResolvesToTheIconHandler() throws Exception {
        assertEquals("getProfileIcon", resolve("GET", PREVIOUSLY_AMBIGUOUS_URI, null));
        assertEquals("getProfileIcon", resolve("GET", PREVIOUSLY_AMBIGUOUS_URI, MediaType.ALL_VALUE));
        assertEquals("getProfileIcon", resolve("GET", PREVIOUSLY_AMBIGUOUS_URI, MediaType.IMAGE_PNG_VALUE));
    }

    @Test
    void getOnThePreviouslyAmbiguousUriRefusesAnythingButAnImage() {
        // The only handler left on this URI produces image/png, so a client asking
        // for JSON is refused by content negotiation instead of being handed the
        // by-user lookup the tie used to give it.
        assertThrows(HttpMediaTypeNotAcceptableException.class,
                () -> resolve("GET", PREVIOUSLY_AMBIGUOUS_URI, MediaType.APPLICATION_JSON_VALUE));
    }

    @Test
    void putOnThePreviouslyAmbiguousUriResolvesToOneHandler() throws Exception {
        assertEquals("updateIcon", resolve("PUT", PREVIOUSLY_AMBIGUOUS_URI, null));
    }

    @Test
    void deleteOnThePreviouslyAmbiguousUriResolvesToOneHandler() throws Exception {
        assertEquals("deleteIcon", resolve("DELETE", PREVIOUSLY_AMBIGUOUS_URI, null));
    }

    @Test
    void theIconLiteralUnderTheByUserPathResolvesToTheByUserHandlers() throws Exception {
        assertEquals("getProfileByUserId", resolve("GET", ICON_LITERAL_UNDER_BY_USER, null));
        assertEquals("updateProfileByUserId", resolve("PUT", ICON_LITERAL_UNDER_BY_USER, null));
        assertEquals("deleteProfileByUserId", resolve("DELETE", ICON_LITERAL_UNDER_BY_USER, null));
        // Patterns are compared before produces, so the longer by-user pattern wins
        // even where the icon handler is the only one that could satisfy the Accept
        // header. The request is then refused as a malformed user id, not routed to
        // an icon.
        assertEquals("getProfileByUserId",
                resolve("GET", ICON_LITERAL_UNDER_BY_USER, MediaType.IMAGE_PNG_VALUE));
    }

    @Test
    void theAddressLiteralUnderTheByUserPathStaysSplitByHttpMethod() throws Exception {
        // This pair does tie on pattern specificity, and never gets that far: the
        // address route maps POST alone, so the method condition separates them.
        assertEquals("getProfileByUserId", resolve("GET", ADDRESS_LITERAL_UNDER_BY_USER, null));
        assertEquals("updateProfileByUserId", resolve("PUT", ADDRESS_LITERAL_UNDER_BY_USER, null));
        assertEquals("deleteProfileByUserId", resolve("DELETE", ADDRESS_LITERAL_UNDER_BY_USER, null));
        assertEquals("addAddress", resolve("POST", ADDRESS_LITERAL_UNDER_BY_USER, null));
    }

    @Test
    void theByUserLookupResolvesToTheByUserHandlers() throws Exception {
        assertEquals("getProfileByUserId", resolve("GET", "/api/profiles/by-user/" + USER_ID, null));
        assertEquals("updateProfileByUserId", resolve("PUT", "/api/profiles/by-user/" + USER_ID, null));
        assertEquals("deleteProfileByUserId", resolve("DELETE", "/api/profiles/by-user/" + USER_ID, null));
    }

    @Test
    @WithMockUser(authorities = "ADMIN")
    void putOnThePreviouslyAmbiguousUriAnswersBadRequestNotServerError() throws Exception {
        mockMvc.perform(multipart(HttpMethod.PUT, PREVIOUSLY_AMBIGUOUS_URI)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer token"))
                .andExpect(status().isBadRequest())
                // Pinned to the cause, not just the status: a missing `icon` part
                // would answer 400 too, and the contract claims this 400 is `user`
                // failing to convert to a profile id.
                .andExpect(result -> assertInstanceOf(MethodArgumentTypeMismatchException.class,
                        result.getResolvedException()));
    }

    @Test
    @WithMockUser(authorities = "ADMIN")
    void deleteOnThePreviouslyAmbiguousUriAnswersBadRequestNotServerError() throws Exception {
        mockMvc.perform(delete(PREVIOUSLY_AMBIGUOUS_URI)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer token"))
                .andExpect(status().isBadRequest())
                .andExpect(result -> assertInstanceOf(MethodArgumentTypeMismatchException.class,
                        result.getResolvedException()));
    }

    @Test
    @WithMockUser(authorities = "ADMIN")
    void getOnThePreviouslyAmbiguousUriAnswersNotAcceptableForJson() throws Exception {
        mockMvc.perform(get(PREVIOUSLY_AMBIGUOUS_URI).accept(MediaType.APPLICATION_JSON))
                .andExpect(status().isNotAcceptable());
    }

    @Test
    @WithMockUser(authorities = "ADMIN")
    void theByUserLookupStillAnswers() throws Exception {
        when(profileService.getProfileByUserId(anyLong()))
                .thenReturn(Profile.builder().id(1L).userId(USER_ID).build());

        mockMvc.perform(get("/api/profiles/by-user/" + USER_ID))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.userId").value((int) USER_ID));
    }
}
