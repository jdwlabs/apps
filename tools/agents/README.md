# 🐳 jdwlabs/agents

![Version](https://img.shields.io/badge/version-1.0.0-blue)

A custom Docker image based on [`ghcr.io/actions/actions-runner`](https://github.com/actions/runner) with additional tools preinstalled:

- ✅ Zulu JDK 21
- ✅ Node.js 24
- ✅ Go 1.26.2
- ✅ Playwright (Chromium, Firefox, WebKit)
- ✅ Cypress
- ✅ Linux GUI deps for headless testing

---

## 🚀 Publishing (Multi-Registry)

To build and push to both **Docker Hub** and **GitHub Container Registry (GHCR)**:

```bash
./publish.sh [version|patch|minor|major]
```

### 💡 Versioning Options

You can provide versioning arguments to `publish.sh` to automatically bump the version stored in the `VERSION` file:

- `patch`  
  → `1.0.0` → `1.0.1`
- `minor`  
  → `1.0.1` → `1.1.0`
- `major`  
  → `1.1.0` → `2.0.0`
- Specific version  
  → `3.2.1` sets it explicitly
- _(default)_  
  → Uses the current version in the `VERSION` file with no bump

---

### 📤 Pushes to:

- Docker Hub: `docker.io/jdwlabs/agents`
- GitHub Container Registry: `ghcr.io/jdwlabs/agents`

---

### 🛠 Requirements

- Docker Buildx enabled
  ```bash
  docker buildx create --use
  ```
- Logged in to Docker Hub
  ```bash
  docker login
  ```
- Logged in to GHCR
  ```bash
  echo $GITHUB_TOKEN | docker login ghcr.io -u <your-gh-username> --password-stdin
  ```

---

## 🧪 Local Development & Debugging

Use `dev.sh` to build, run, or debug the image locally.

```bash
./dev.sh [command]
```

### 🔧 Commands

| Command | Description                         |
| ------- | ----------------------------------- |
| build   | Build the image locally             |
| run     | Run the container interactively     |
| rebuild | Build and run (clean start)         |
| shell   | Exec into a running container shell |
| clean   | Remove container + image            |
| help    | Show available commands             |

Example:

```bash
./dev.sh rebuild
```

---

## 📁 Project Layout

```
.
├── Dockerfile       # Base image definition
├── VERSION          # Semantic version tracking
├── publish.sh       # CI-friendly build & push script
├── dev.sh           # Local development helper
└── README.md        # You're here!
```

---

## 🧰 Tools Available in Image

- Java: `java -version`
- Node.js: `node -v`, `npm -v`, `pnpm -v`
- Go: `go version`
- Playwright: `playwright --version`
- Cypress: `cypress version`

---

## 📦 Platforms Supported

- `linux/amd64`

---
