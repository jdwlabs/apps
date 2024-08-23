package com.jdw.usersrole;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertThrows;

@Tag("fast")
@Tag("integration")
@SpringBootTest
class UsersRoleApplicationTests {
	@Test
	void contextLoads() {
	}

	@Test
	void mainTest() {
		assertDoesNotThrow(() -> UsersRoleApplication.main(new String[] {}));
	}
}
