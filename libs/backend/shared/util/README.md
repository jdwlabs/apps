# Go Shared Util

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

The **Go Shared Util** library provides reusable utility functions and middleware for Go services in the JDW platform.
It includes environment helpers and HTTP logging middleware that can be shared across Go-based microservices and
libraries.

---

## 📁 Project Structure

```
libs/backend/shared/util/
├── go.mod                     # Go module definition
├── util.go                    # Shared utility functions and middleware
├── util_test.go               # Unit tests for the util package
├── project.json               # Nx project configuration
```

---

## 🔧 Usage

### GetEnvOrDefault

Retrieve an environment variable or fallback to a default value if not set or empty.

```go
  import "jdw/libs/backend/shared/util"

  port := util.GetEnvOrDefault("PORT", "8080")
```

### HTTP Logging Middleware

Log details about incoming HTTP requests, including status, method, duration, IP, and user agent using `slog`.

```go
  import (
    "net/http"
    "jdw/libs/backend/shared/util"
  )

  func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello, World!"))
  })

  wrapped := util.Logging(mux)
  http.ListenAndServe(":8080", wrapped)
}
```

---

## 🧪 Testing

Run unit tests for the package:

```bash
nx test go-shared-util
```

---

## 🛠 Linting

Check for linting issues using:

```bash
nx lint go-shared-util
```

---

## 🧹 Go Mod Tidy

Clean up and verify dependencies with:

```bash
nx tidy go-shared-util
```

---

## 📌 Notes

- This library is intended for shared use across Go services and is kept framework-agnostic.
- Logging middleware is designed for use with standard `net/http` and structured logging via `slog`.
- The library is maintained as part of a modular Nx monorepo to ensure consistent tooling and CI/CD pipelines across
  languages.

---

## 📚 Related Packages

- [`servicediscovery`](../../../../apps/backend/servicediscovery): The Go service currently importing this package.
- [`angular-usersui-util`](../../../frontend/usersui/util): Utility library for the Angular Users UI.
