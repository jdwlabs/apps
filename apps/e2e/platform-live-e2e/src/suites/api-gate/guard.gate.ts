import { test, expect } from '../../harness/fixtures';
import { GATE, PRD_SAFE } from '../../harness/tags';

test(
  'the harness resolved a profile',
  { tag: [GATE, PRD_SAFE] },
  async ({ profile }) => {
    expect(profile.apiBaseUrl).toMatch(/^https:\/\//);
  },
);
