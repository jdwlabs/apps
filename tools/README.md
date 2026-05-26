# Tools Directory

The `tools/` directory contains **Nx-adjacent** automation only — code that requires Nx context to run (custom executors, generators, and Docker-based dev agents).

## Structure

```
tools/
└── agents/
    ├── dev.sh         # Start the local Docker development agent
    ├── publish.sh     # Publish agent artifacts
    ├── Dockerfile     # Docker image for the dev agent
    ├── README.md      # Agent-specific documentation
    └── VERSION        # Agent version file
```

## Non-Nx Scripts

Shell scripts, Docker Compose files, and other non-Nx helpers live in [`scripts/`](../scripts/README.md).
