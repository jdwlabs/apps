## 1.0.2 (2026-07-22)

This was a version bump only for ai-sre-relay to align it with other projects, there were no code changes.

## 1.0.1 (2026-07-20)

### 🩹 Fixes

- **ai-sre-relay:** dedup alert tickets and sanitize llm analysis before filing ([05ef4893](https://github.com/jdwlabs/apps/commit/05ef4893))

### ❤️ Thank You

- Claude Fable 5
- Jake Willmsen @jdwillmsen

# 1.0.0 (2026-07-10)

### 🚀 Features

- **ai-sre-relay:** render Discord alerts as rich embeds ([9d782844](https://github.com/jdwlabs/apps/commit/9d782844))
- **ai-sre-relay:** bound concurrency, drain on shutdown, auth+limits ([d7d75a93](https://github.com/jdwlabs/apps/commit/d7d75a93))
- **ai-sre-relay:** add container images ([3fd53ad8](https://github.com/jdwlabs/apps/commit/3fd53ad8))
- **ai-sre-relay:** add webhook ingress and wire main ([dcbb56b1](https://github.com/jdwlabs/apps/commit/dcbb56b1))
- **ai-sre-relay:** add pipeline orchestrator with independence tests ([9a293fc2](https://github.com/jdwlabs/apps/commit/9a293fc2))
- **ai-sre-relay:** add github pr opener ([f423ef55](https://github.com/jdwlabs/apps/commit/f423ef55))
- **ai-sre-relay:** add jira upsert with fingerprint dedup ([38fd20ae](https://github.com/jdwlabs/apps/commit/38fd20ae))
- **ai-sre-relay:** add discord notifier ([41816b6b](https://github.com/jdwlabs/apps/commit/41816b6b))
- **ai-sre-relay:** add litellm patch generator with confidence gate ([3448f2c2](https://github.com/jdwlabs/apps/commit/3448f2c2))
- **ai-sre-relay:** add holmes investigation client ([a5660cb6](https://github.com/jdwlabs/apps/commit/a5660cb6))
- **ai-sre-relay:** add domain types ([d73db1e0](https://github.com/jdwlabs/apps/commit/d73db1e0))
- **ai-sre-relay:** scaffold nx go project with healthz ([2544cc27](https://github.com/jdwlabs/apps/commit/2544cc27))

### 🩹 Fixes

- **ai-sre-relay:** reopen done jira issues instead of duplicating ([c5072ddf](https://github.com/jdwlabs/apps/commit/c5072ddf))
- **ai-sre-relay:** truncate patch rationale to stay under discord's field limit ([2aadfdd7](https://github.com/jdwlabs/apps/commit/2aadfdd7))
- **ai-sre-relay:** configurable jira issue type, default Task ([2044b870](https://github.com/jdwlabs/apps/commit/2044b870))
- **ai-sre-relay:** failure notice survives dead per-alert context ([75498899](https://github.com/jdwlabs/apps/commit/75498899))
- **ai-sre-relay:** migrate jira dedup search to /search/jql ([f49dda49](https://github.com/jdwlabs/apps/commit/f49dda49))
- **ai-sre-relay:** drop client timeout on LLM-bound calls ([8db9f384](https://github.com/jdwlabs/apps/commit/8db9f384))
- **ai-sre-relay:** call Holmes /api/chat; investigate endpoint gone ([ff0194da](https://github.com/jdwlabs/apps/commit/ff0194da))
- **ai-sre-relay:** check jira search status, recover webhook panics, harden github/patch ([212ec5b0](https://github.com/jdwlabs/apps/commit/212ec5b0))
- **ai-sre-relay:** fail on jira decode/marshal errors and test error+auth paths ([64ff7a03](https://github.com/jdwlabs/apps/commit/64ff7a03))
- **ai-sre-relay:** use generic example in IssueKey comment ([a39e083f](https://github.com/jdwlabs/apps/commit/a39e083f))
- **ai-sre-relay:** remove build artifact and comment refs, assert healthz body ([b39625e4](https://github.com/jdwlabs/apps/commit/b39625e4))

### ❤️ Thank You

- Claude Fable 5
- Claude Opus 4.8
- Claude Sonnet 4.6
- Claude Sonnet 5
- Jake Willmsen @jdwillmsen

## 0.2.1 (2026-07-07)

### 🩹 Fixes

- **ai-sre-relay:** reopen done jira issues instead of duplicating ([18173894](https://github.com/jdwlabs/apps/commit/18173894))

### ❤️ Thank You

- Jake Willmsen @jdwillmsen

## 0.2.0 (2026-07-07)

### 🚀 Features

- **ai-sre-relay:** render Discord alerts as rich embeds ([8ac68d07](https://github.com/jdwlabs/apps/commit/8ac68d07))

### 🩹 Fixes

- **ai-sre-relay:** truncate patch rationale to stay under discord's field limit ([4f58859d](https://github.com/jdwlabs/apps/commit/4f58859d))

### ❤️ Thank You

- Claude Sonnet 5
- Jake Willmsen @jdwillmsen

# Changelog

This file was generated using [@jscutlery/semver](https://github.com/jscutlery/semver).

## [0.1.5](https://github.com/jdwlabs/apps/compare/ai-sre-relay-0.1.4...ai-sre-relay-0.1.5) (2026-07-04)

### Bug Fixes

- **ai-sre-relay:** configurable jira issue type, default Task ([54d0c74](https://github.com/jdwlabs/apps/commit/54d0c7446b68a87608f7ea73a0fedd053bd1a6c4))

## [0.1.4](https://github.com/jdwlabs/apps/compare/ai-sre-relay-0.1.3...ai-sre-relay-0.1.4) (2026-07-04)

### Bug Fixes

- **ai-sre-relay:** failure notice survives dead per-alert context ([7066f85](https://github.com/jdwlabs/apps/commit/7066f859ace3a8b1f136d5088f6ea979b706888c))

## [0.1.3](https://github.com/jdwlabs/apps/compare/ai-sre-relay-0.1.2...ai-sre-relay-0.1.3) (2026-07-04)

### Bug Fixes

- **ai-sre-relay:** migrate jira dedup search to /search/jql ([4e94c3e](https://github.com/jdwlabs/apps/commit/4e94c3ed2d991584de7afa31c2b2e695c009dc4b))

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
