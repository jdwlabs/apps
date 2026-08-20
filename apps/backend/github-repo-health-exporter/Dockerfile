# Distroless final image; consumes the Nx build output at ./dist/...
# The CI build-image target runs after `nx build`, so dist/ is populated.
FROM gcr.io/distroless/base-debian12:nonroot AS runner

WORKDIR /app
COPY ./dist/apps/backend/github-repo-health-exporter/github-repo-health-exporter .

USER nonroot:nonroot
EXPOSE 9090
ENTRYPOINT ["/app/github-repo-health-exporter"]
