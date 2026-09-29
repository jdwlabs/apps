FROM node:24-bookworm-slim
ARG PLAYWRIGHT_VERSION
ARG OPENAPI_FETCH_VERSION
WORKDIR /e2e
# Exact versions come from the workspace lockfile at build time, so the image
# runs the same Playwright the repo's tests were written against.
RUN test -n "$PLAYWRIGHT_VERSION" && test -n "$OPENAPI_FETCH_VERSION" \
 && npm init -y >/dev/null \
 && npm install --no-audit --no-fund "@playwright/test@${PLAYWRIGHT_VERSION}" "openapi-fetch@${OPENAPI_FETCH_VERSION}" \
 && npm cache clean --force
COPY apps/e2e/platform-live-e2e/playwright.config.ts ./
COPY apps/e2e/platform-live-e2e/src ./src
ENV E2E_REPORT_DIR=/e2e/report
RUN mkdir -p /e2e/report && chown node /e2e/report
USER node
ENTRYPOINT ["npx", "playwright", "test", "--config", "playwright.config.ts", "--project", "api-gate"]
