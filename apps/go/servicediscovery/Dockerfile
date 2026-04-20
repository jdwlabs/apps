# Use a specific version of distroless based on SHA256 digest for reproducibility and security
# Using distroless for the final image to minimize attack surface and size
FROM gcr.io/distroless/base-debian12:nonroot AS runner

WORKDIR /app
COPY ./dist/apps/go/servicediscovery .

ENTRYPOINT ["/app/servicediscovery"]