import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { CONTRACTS, renderContractTypes } from './contract-types.mjs';

describe('renderContractTypes', () => {
  it('renders one module per frozen contract', async () => {
    const rendered = await renderContractTypes();
    expect(Object.keys(rendered).sort()).toEqual(['identity', 'profile']);
  });

  it('matches the committed generated files', async () => {
    const rendered = await renderContractTypes();
    for (const name of Object.keys(CONTRACTS)) {
      const committed = readFileSync(
        join(__dirname, '../src/harness/generated', `${name}.ts`),
        'utf8',
      );
      expect(rendered[name]).toBe(committed);
    }
  });

  it('types the login response token', async () => {
    const rendered = await renderContractTypes();
    expect(rendered['identity']).toContain('jwtToken: string');
  });
});
