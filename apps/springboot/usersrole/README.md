# Users Role

The UsersRole application is a Spring Boot service designed for managing user roles in a system.
It handles CRUD operations for roles and users and integrates with a PostgreSQL database. This
application is part of a larger project that includes an Angular frontend and various libraries
for shared components and utilities.

## Project Structure

```markdown
.
├── build.gradle.kts
├── CHANGELOG.md
├── gradlew
├── gradlew.bat
├── project.json
├── settings.gradle.kts
└── src
├── main
│   ├── java
│   └── resources
└── test
└── java
```

## Environment Variables

```properties
UR_JWT_SECRET_KEY=<JWT SECRET KEY>
UR_JWT_EXPIRATION_TIME_MS=<JWT EXPIRATION TIME>
UR_PG_DATASOURCE_URL=<JDBC URL>
UR_PG_USERNAME=<POSTGRES USERNAME>
UR_PG_PASSWORD=<POSTGRES PASSWORD>
```

## Running Tests

To run the tests use the `nx test target`

```shell
npx nx test usersrole
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
