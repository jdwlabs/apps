package com.jdw.usersrole.serialization;

import com.jayway.jsonpath.DocumentContext;
import com.jayway.jsonpath.JsonPath;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.server.LocalServerPort;
import org.springframework.web.client.RestClient;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;

@Tag("fast")
@Tag("integration")
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
class OpenApiPasswordContractTests {
    @LocalServerPort
    private int port;

    @Test
    void userSchemaDoesNotPublishThePasswordHash() {
        DocumentContext apiDocs = apiDocs();

        Map<String, Object> userProperties = apiDocs.read("$.components.schemas.User.properties");

        assertNotNull(userProperties);
        assertFalse(userProperties.containsKey("password"),
                "the served OpenAPI document still carries User.password: " + userProperties.keySet());
    }

    @Test
    void userRequestSchemaMarksThePasswordWriteOnly() {
        DocumentContext apiDocs = apiDocs();

        assertEquals(Boolean.TRUE, apiDocs.read("$.components.schemas.UserRequestDTO.properties.password.writeOnly"));
    }

    private DocumentContext apiDocs() {
        String body = RestClient.create()
                .get()
                .uri("http://localhost:" + port + "/openapi/api-docs")
                .retrieve()
                .body(String.class);
        return JsonPath.parse(body);
    }
}
