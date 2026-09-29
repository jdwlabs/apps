import openapiTS, { astToString } from 'openapi-typescript';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const workspaceRoot = join(here, '../../../..');
const contractsDir = join(
  workspaceRoot,
  'apps/backend/usersrole/docs/contracts',
);
const generatedDir = join(here, '../src/harness/generated');

export const CONTRACTS = {
  identity: 'identity-service.openapi.yaml',
  profile: 'profile-service.openapi.yaml',
};

const HEADER =
  '// Generated from the frozen usersrole contracts. Regenerate with\n// `pnpm exec nx run platform-live-e2e:generate-contract-types`; never edit by hand.\n\n';

export async function renderContractTypes() {
  const rendered = {};
  for (const [name, file] of Object.entries(CONTRACTS)) {
    const ast = await openapiTS(pathToFileURL(join(contractsDir, file)));
    rendered[name] = HEADER + astToString(ast);
  }
  return rendered;
}

async function main() {
  const check = process.argv.includes('--check');
  const rendered = await renderContractTypes();
  const stale = [];
  mkdirSync(generatedDir, { recursive: true });
  for (const [name, content] of Object.entries(rendered)) {
    const target = join(generatedDir, `${name}.ts`);
    if (check) {
      let committed = '';
      try {
        committed = readFileSync(target, 'utf8');
      } catch {
        // A missing file is drift like any other.
      }
      if (committed !== content) stale.push(target);
    } else {
      writeFileSync(target, content);
    }
  }
  if (stale.length > 0) {
    console.error(
      `Generated contract types are stale:\n  ${stale.join('\n  ')}\nRun: pnpm exec nx run platform-live-e2e:generate-contract-types`,
    );
    process.exit(1);
  }
}

if (import.meta.url === pathToFileURL(process.argv[1]).href) {
  await main();
}
