# Distroless final image; consumes the Nx build output at ./dist/...
# The CI build-image target runs after `nx build`, so dist/ is populated.
FROM gcr.io/distroless/base-debian12:nonroot AS runner

WORKDIR /app
COPY ./dist/apps/backend/ai-sre-relay/ai-sre-relay .

EXPOSE 8080
ENTRYPOINT ["/app/ai-sre-relay"]
