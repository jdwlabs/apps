package com.jdw.usersrole;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.TestPropertySource;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;

@Tag("fast")
@Tag("integration")
@SpringBootTest
@TestPropertySource(properties = "server.port=0")
class UsersRoleApplicationTests {
    @Test
    void contextLoads() {
    }

    @Test
    void mainTest() {
        assertDoesNotThrow(() -> UsersRoleApplication.main(new String[]{}));
    }
}
