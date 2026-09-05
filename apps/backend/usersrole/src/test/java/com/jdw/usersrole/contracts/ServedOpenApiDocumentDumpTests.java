package com.jdw.usersrole.contracts;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.server.LocalServerPort;
import org.springframework.web.client.RestClient;

import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

/**
 * Writes the springdoc document this build serves to a file so the frozen
 * contracts can be diffed against it. The output path is deliberately a build
 * directory: the served document is evidence, not a committed artifact.
 */
@Tag("fast")
@Tag("integration")
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
class ServedOpenApiDocumentDumpTests {
    private static final String OUTPUT_PATH_PROPERTY = "usersrole.openapi.dump";
    private static final String DEFAULT_OUTPUT_PATH = "build/openapi/served-api-docs.json";

    @LocalServerPort
    private int port;

    @Test
    void dumpsTheServedDocument() throws Exception {
        String body = RestClient.create()
                .get()
                .uri("http://localhost:" + port + "/openapi/api-docs")
                .retrieve()
                .body(String.class);

        assertNotNull(body);
        assertTrue(body.contains("\"paths\""), "served document has no paths object");

        Path output = Paths.get(System.getProperty(OUTPUT_PATH_PROPERTY, DEFAULT_OUTPUT_PATH));
        Files.createDirectories(output.toAbsolutePath().getParent());
        Files.writeString(output, body, StandardCharsets.UTF_8);
    }
}
