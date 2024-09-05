package com.jdw.usersrole;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;

@Tag("fast")
@Tag("integration")
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
class UsersRoleApplicationTests {
    @Test
    void contextLoads() {
    }

    @Test
    void mainTest() {
        String[] args = {"--server.port=0"};
        assertDoesNotThrow(() -> UsersRoleApplication.main(args));
    }
}
