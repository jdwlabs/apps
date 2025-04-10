# Docker Configuration

The **docker** directory contains the configuration and orchestration files required to run the JDW platform services
locally using Docker Compose v2. It includes configurations for the containerized services, such as authentication, user
management, roles, and service discovery.

---

## 📁 Project Structure

```
docker/
├── compose.yaml                # Docker Compose configuration file to run services
└── config.json                 # Configuration file for service discovery
```

---

## 🔧 Usage

To set up and run the JDW platform locally using Docker, use the provided docker compose configuration. This will spin
up several microservices required for the platform, including:

- **jdw-container**: The main container for the JDW platform.
- **jdw-authui**: The authentication UI service.
- **jdw-authdb**: PostgreSQL database for authentication.
- **jdw-usersrole**: The service managing users and roles.
- **jdw-servicediscovery**: Service discovery service for registering and locating microservices.
- **jdw-usersui**: The user management UI service.
- **jdw-rolesui**: The roles management UI service.

### Running Services with Docker Compose v2

1. Ensure you have Docker v20.10.0 or higher installed (which includes Docker Compose v2).
2. From the root of the project, run the following command to start all services:

```
docker compose -f tools/docker/compose.yaml up
```

This will start the services as defined in the `compose.yaml` file and map the respective ports:

- **jdw-container**: Accessible at `http://localhost:4200`
- **jdw-authui**: Accessible at `http://localhost:4201`
- **jdw-usersui**: Accessible at `http://localhost:4202`
- **jdw-rolesui**: Accessible at `http://localhost:4203`
- **jdw-authdb**: PostgreSQL database running on `jdbc:postgresql://localhost:5432`
- **jdw-servicediscovery**: Service discovery accessible at `http://localhost:9000`

### Stopping Services

To stop all running services, use the following command:

```
docker compose -f tools/docker/compose.yaml down
```

This will stop and remove all running containers.

---

## 🔧 Configuration

The **config.json** file contains the configuration details for the **jdw-servicediscovery** service. It includes the
mappings for the microfrontends and their respective URLs and metadata. Below is a sample configuration structure:

```
{
  "remotes": {
    "authui": "http://localhost:4201",
    "usersui": "http://localhost:4202"
  },
  "microFrontends": [
    {
      "name": "authui",
      "path": "auth",
      "remoteName": "authui",
      "moduleName": "./Routes",
      "url": "http://localhost:4201",
      "icon": "login",
      "title": "Auth",
      "description": "This contains sign in and sign up functionality"
    },
    {
      "name": "usersui",
      "path": "users",
      "remoteName": "usersui",
      "moduleName": "./Routes",
      "url": "http://localhost:4202",
      "icon": "groups",
      "title": "Users",
      "description": "This contains viewing users and managing profiles functionality"
    },
    {
      "name": "rolesui",
      "path": "roles",
      "remoteName": "rolesui",
      "moduleName": "./Routes",
      "url": "http://localhost:4203",
      "icon": "lock",
      "title": "Roles",
      "description": "This contains viewing and managing roles functionality"
    }
  ]
}
```

---

## 📌 Notes

- The docker compose configuration is optimized for local development and testing. For production environments, adjust
  the configuration as needed, especially for scaling and security.
- The **jdw-servicediscovery** service allows for dynamic discovery of microfrontends and enables smooth routing between
  them.
- The **authdb** service uses persistent storage with Docker volumes to ensure that the PostgreSQL data persists between
  container restarts.

---

## 🧪 Testing

Ensure all services are up and running before performing any manual or automated tests. Access each service by
navigating to their respective URLs as outlined above.

