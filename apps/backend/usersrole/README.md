# UsersRole

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/usersrole)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/usersrole)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwlabs/usersrole?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**UsersRole** is a Spring Boot service designed to manage user roles within the JDW Platform. It handles CRUD operations
for roles and users and integrates with a PostgreSQL database. This service is part of a larger ecosystem that includes
an Angular frontend and various shared libraries for components and utilities.

---

## 📁 Project Structure

```
apps/backend/usersrole/
├── build.gradle.kts                                   # Gradle build configuration
├── .gitignore                                         # Git ignore rules
├── gradle/                                            # Gradle wrapper files
├── gradlew                                            # Unix Gradle wrapper script
├── gradlew.bat                                        # Windows Gradle wrapper script
├── project.json                                       # Nx project definition and targets
├── README.md                                          # This README file
├── settings.gradle.kts                                # Gradle settings
└── src
    ├── main
    │   ├── java
    │   │   └── com
    │   │       └── jdw
    │   │           └── usersrole
    │   │               ├── configs                    # Security and JWT configurations
    │   │               ├── controllers                # REST API controllers
    │   │               ├── daos                       # Data access objects with Postgres implementations
    │   │               ├── dtos                       # Request and response DTOs
    │   │               ├── exceptions                 # Application exceptions
    │   │               ├── metrics                    # Metrics and performance logging
    │   │               ├── models                     # Domain models
    │   │               ├── repositories               # Repository implementations
    │   │               ├── services                   # Business logic and service layer
    │   │               └── UsersRoleApplication.java
    │   └── resources
    │       └── application.yaml                       # Application configuration
    └── test                                           # Unit and integration tests
```

---

## 🔧 Environment Variables

Define the following environment variables to configure the UsersRole service:

```properties
UR_JWT_SECRET_KEY=<JWT SECRET KEY>
UR_JWT_EXPIRATION_TIME_MS=<JWT EXPIRATION TIME>
UR_PG_DATASOURCE_URL=<JDBC URL>
UR_PG_USERNAME=<POSTGRES USERNAME>
UR_PG_PASSWORD=<POSTGRES PASSWORD>
```

---

## 🚀 Running the Application

You can run the application using Gradle or through your preferred IDE. For a quick test from the command line:

```shell
./gradlew bootRun
```

The service will start with the configurations defined in `application.yaml`.

---

## 🧪 Running Tests

Run the tests using the Nx target:

```shell
npx nx test usersrole
```

This command executes the unit and integration tests for the UsersRole service.

---

## 📦 Deployment

UsersRole is deployed as part of the JDW Platform services. Ensure that the correct environment variables and PostgreSQL
connection configurations are provided during deployment.
