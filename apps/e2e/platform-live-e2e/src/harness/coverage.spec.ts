import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { parse } from 'yaml';

const root = join(__dirname, '../../../../..');
const contracts = [
  'identity-service.openapi.yaml',
  'profile-service.openapi.yaml',
];
const suitesDir = join(__dirname, '../suites/api-gate');

function contractOperationIds(): string[] {
  const ids: string[] = [];
  for (const file of contracts) {
    const doc = parse(
      readFileSync(
        join(root, 'apps/backend/usersrole/docs/contracts', file),
        'utf8',
      ),
    );
    for (const methods of Object.values(
      doc.paths as Record<string, Record<string, { operationId?: string }>>,
    )) {
      for (const op of Object.values(methods))
        if (op?.operationId) ids.push(op.operationId);
    }
  }
  return ids;
}

function declaredOperationIds(): Set<string> {
  const declared = new Set<string>();
  for (const file of readdirSync(suitesDir).filter((f) =>
    f.endsWith('.gate.ts'),
  )) {
    const source = readFileSync(join(suitesDir, file), 'utf8');
    for (const call of source.matchAll(/operations\(([^)]*)\)/g)) {
      for (const id of call[1].matchAll(/'([A-Za-z]+)'/g)) declared.add(id[1]);
    }
  }
  return declared;
}

describe('api-gate coverage', () => {
  it('reads all 33 operations from the frozen contracts', () => {
    // Stated separately so a parser that silently stops matching cannot let
    // the next assertion pass against an empty list.
    expect(contractOperationIds()).toHaveLength(33);
  });

  it('exercises every contract operation', () => {
    const declared = declaredOperationIds();
    expect(contractOperationIds().filter((id) => !declared.has(id))).toEqual(
      [],
    );
  });

  it('declares no operation the contracts do not have', () => {
    const known = new Set(contractOperationIds());
    expect([...declaredOperationIds()].filter((id) => !known.has(id))).toEqual(
      [],
    );
  });
});
