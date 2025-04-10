# Tools Directory

The **tools** directory contains various scripts and configuration files used to automate and manage workflows in the
JDW platform. These tools help with versioning, Docker setup, configuration management, and more, facilitating
streamlined development, testing, and deployment processes across the platform.

---

## 📁 Project Structure

```
tools/
├── agents/
│   ├── dev.sh                   # Script to start a development agent
│   ├── Dockerfile                # Dockerfile for the development agent
│   ├── publish.sh               # Script to publish agent artifacts
│   ├── README.md                # README file for agents folder
│   └── VERSION                  # Version file for the agent
├── create-version.sh            # Script to create a new version
├── docker/
│   ├── compose.yaml             # Docker Compose configuration for local services
│   └── config.json              # Docker-related configuration file
├── format.sh                    # Script to format codebase
├── prepare-config.sh            # Script to prepare configuration files
├── restore-config.sh            # Script to restore configuration files
├── update-app.sh                # Script to update the application version
└── update-version.sh            # Script to update versioning info
```

---

## 🔧 Usage

### **Development Agent Scripts**

The `agents` folder contains scripts that help manage development agents and related tasks.

- `dev.sh`: Starts a local development agent environment.
- `publish.sh`: Publishes artifacts for the agent.
- `Dockerfile`: Contains Docker setup for the agent.

Example usage:

```bash
# Start the development agent environment
./tools/agents/dev.sh
```

### **Versioning and Updates**

- `create-version.sh`: Creates a new version for the platform.
- `update-version.sh`: Updates versioning information across the repository.
- `update-app.sh`: Updates the application version to the latest version.

Example usage:

```bash
# Create a new version for the platform
./tools/create-version.sh
```

### **Docker Management**

- `docker/compose.yaml`: Configuration for setting up local services with Docker Compose.
- `docker/config.json`: Docker-related configuration, including service settings.

To run the Docker Compose setup:

```bash
docker-compose -f tools/docker/compose.yaml up
```

### **Configuration Management**

- `prepare-config.sh`: Prepares configuration files for the platform.
- `restore-config.sh`: Restores configuration files from backups.

Example usage:

```bash
# Prepare platform configuration
./tools/prepare-config.sh

# Restore configuration
./tools/restore-config.sh
```

### **Code Formatting**

- `format.sh`: Formats the entire codebase according to the project's coding standards.

Run this script to format the code:

```bash
./tools/format.sh
```

---

## 🧪 Testing

To ensure the scripts and tools are working as expected, consider running the respective scripts in a controlled
environment and verify the outputs.

---

## 🛠 Linting

To ensure consistency and maintain best practices in the scripts, consider adding linting tools and running them before
pushing changes. This will help with script quality and readability.

---

## 📌 Notes

- The **tools** directory is intended to house all automation and configuration-related scripts for the JDW platform.
- The scripts are modular, allowing for specific tasks like versioning, Docker orchestration, or development environment
  management.
- They are designed to work in conjunction with the CI/CD pipeline to facilitate smooth development and deployment
  workflows.



