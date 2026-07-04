# Changelog

This file was generated using [@jscutlery/semver](https://github.com/jscutlery/semver).

## [0.1.2](https://github.com/jdwlabs/apps/compare/ai-sre-relay-0.1.1...ai-sre-relay-0.1.2) (2026-07-04)

### Bug Fixes

- **ai-sre-relay:** drop client timeout on LLM-bound calls ([bca27ee](https://github.com/jdwlabs/apps/commit/bca27eebf888e36138c94ae1660be213f997059b))

## [0.1.1](https://github.com/jdwlabs/apps/compare/ai-sre-relay-0.1.0...ai-sre-relay-0.1.1) (2026-07-04)

### Bug Fixes

- **ai-sre-relay:** call Holmes /api/chat; investigate endpoint gone ([6de9d11](https://github.com/jdwlabs/apps/commit/6de9d11d0493fa3b61185898ee509bcc46f8ad44))

## 0.1.0 (2026-07-03)

### Features

- **ai-sre-relay:** add container images ([68c88f4](https://github.com/jdwlabs/apps/commit/68c88f4bc5dca656e81e7d0c6500c4db8dbaa82f))
- **ai-sre-relay:** add discord notifier ([91cd7a5](https://github.com/jdwlabs/apps/commit/91cd7a5a2e3be382171f31ccfeef5654249f60db))
- **ai-sre-relay:** add domain types ([4e2a107](https://github.com/jdwlabs/apps/commit/4e2a1073cb81fb13adb92068eba6288100876b5e))
- **ai-sre-relay:** add github pr opener ([4e41bde](https://github.com/jdwlabs/apps/commit/4e41bde5dc5664c4f89d14bf3230a3f0f1a78631))
- **ai-sre-relay:** add holmes investigation client ([ee76c71](https://github.com/jdwlabs/apps/commit/ee76c71bfce0cdccec573aa4be2c676fb3b77d93))
- **ai-sre-relay:** add jira upsert with fingerprint dedup ([0ffb3a6](https://github.com/jdwlabs/apps/commit/0ffb3a6f9921d3beb94e590e29883046ac5822ca))
- **ai-sre-relay:** add litellm patch generator with confidence gate ([2dd6626](https://github.com/jdwlabs/apps/commit/2dd66266bd31bbc0c68c1b554f5e3e654fe6f461))
- **ai-sre-relay:** add pipeline orchestrator with independence tests ([d4729b0](https://github.com/jdwlabs/apps/commit/d4729b0778748e2f0aaafa4b9c9c5bd3be7f0211))
- **ai-sre-relay:** add webhook ingress and wire main ([9b17023](https://github.com/jdwlabs/apps/commit/9b17023d5c495ec9c1c64b53dcd6c7ed151a31a1))
- **ai-sre-relay:** bound concurrency, drain on shutdown, auth+limits ([b1f915a](https://github.com/jdwlabs/apps/commit/b1f915a85c996dd8ae9f7541bb7cfee61aebf85e))
- **ai-sre-relay:** scaffold nx go project with healthz ([ac4b27d](https://github.com/jdwlabs/apps/commit/ac4b27d3a07d85b6728c965165c1dd6570818aa4))

### Bug Fixes

- **ai-sre-relay:** check jira search status, recover webhook panics, harden github/patch ([4e3f310](https://github.com/jdwlabs/apps/commit/4e3f3105a640c7c5138242e7e2dfd3c197bc8e68))
- **ai-sre-relay:** fail on jira decode/marshal errors and test error+auth paths ([9f5db1f](https://github.com/jdwlabs/apps/commit/9f5db1f97073fc982e00a9677471753b0be2db1b))
- **ai-sre-relay:** remove build artifact and comment refs, assert healthz body ([892af28](https://github.com/jdwlabs/apps/commit/892af281043368c767b930e3876bccd94d3a5449))
- **ai-sre-relay:** use generic example in IssueKey comment ([b3e3e56](https://github.com/jdwlabs/apps/commit/b3e3e56151c7b800480405c63d6b175e320e13c2))
