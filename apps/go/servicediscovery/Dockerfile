# Remember to check for updates to base images to incorporate security patches
# Use a specific version of golang based on SHA256 digest for reproducibility and security
FROM golang:1.23 AS builder-go

# Use a specific version of node base on SHA256 digest for reproducibility and security
FROM node:lts AS builder
WORKDIR /app

ARG GOLANG_VERSION=1.23
COPY --from=builder-go /usr/local/go /usr/local/go
ENV PATH=$PATH:/usr/local/go/bin

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npx nx build servicediscovery

# Use a specific version of distroless based on SHA256 digest for reproducibility and security
# Using distroless for the final image to minimize attack surface and size
FROM gcr.io/distroless/base-debian12:nonroot AS runner

WORKDIR /app
COPY --from=builder /app/dist/apps/go/servicediscovery .

ENTRYPOINT ["/app/servicediscovery"]