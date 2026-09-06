# Distroless final image; consumes the Nx build output at ./dist/...
# The CI build-image target runs after `nx build`, so dist/ is populated.
FROM gcr.io/distroless/base-debian12:nonroot AS runner

WORKDIR /app
COPY ./dist/apps/backend/profile-service/profile-service .

# Numeric rather than the `nonroot` name the base tag already carries. A
# Kubernetes runAsNonRoot check has to resolve a named user against the image's
# /etc/passwd, and a previous deployment here failed that check on an image whose
# USER was a name; a uid needs no resolution.
USER 65532:65532

EXPOSE 8080
ENTRYPOINT ["/app/profile-service"]
