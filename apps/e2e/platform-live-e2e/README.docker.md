# platform-e2e-api

Live API gate for the jdwlabs platform: Playwright `api-gate` suite exercising every frozen-contract operation against a deployed environment.

## Run

    docker run --rm \
      -e E2E_PROFILE=non-public \
      -e E2E_USER_EMAIL -e E2E_USER_PASSWORD \
      -e E2E_ADMIN_EMAIL -e E2E_ADMIN_PASSWORD \
      jdwlabs/platform-e2e-api:<version>

`E2E_PROFILE` is one of `non-public`, `prd-public`, `non-incluster`, `prd-incluster`. prd profiles run only `@prd-safe` tests and never create users. Reports are written to `/e2e/report`.
