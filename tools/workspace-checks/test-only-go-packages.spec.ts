import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, sep } from 'node:path';

/**
 * Go has no way to say "this package is for tests only". A build tag would hide
 * the package from the tests that need it, and an unexported API cannot cross a
 * module boundary at all, so the rule is asserted here instead.
 *
 * `authtest` mints tokens. Anything that ships and can mint its own tokens can
 * mint itself any principal it likes, which is the entire authorization model
 * gone — including the verifier that is supposed to be checking them.
 *
 * This lives in workspace-checks rather than beside the Go code because the rule
 * spans every module: the offending import would sit in a service, and a test
 * inside the library is only selected when the library itself changes. Here it
 * runs whenever anything does, and `cache: false` keeps it from replaying a pass
 * over a tree it never read.
 */
const TEST_ONLY_PACKAGES = ['libs/backend/shared/auth/authtest'];

/** Directories that hold no first-party Go source worth scanning. */
const SKIP = new Set([
  'node_modules',
  '.git',
  '.nx',
  'dist',
  'build',
  'coverage',
  'vendor',
  '.angular',
  'tmp',
]);

const repoRoot = join(__dirname, '..', '..');

const goSourceFiles = (dir: string): string[] => {
  const found: string[] = [];
  for (const entry of readdirSync(dir)) {
    if (SKIP.has(entry)) continue;
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) {
      found.push(...goSourceFiles(path));
      continue;
    }
    // _test.go files are exactly who is allowed to import these packages.
    if (entry.endsWith('.go') && !entry.endsWith('_test.go')) {
      found.push(path);
    }
  }
  return found;
};

describe('test-only Go packages stay out of shipped code', () => {
  it.each(TEST_ONLY_PACKAGES)('nothing that ships imports %s', (pkg) => {
    const owningDir = join(repoRoot, pkg);
    const offenders = goSourceFiles(repoRoot)
      // The package's own files are not importing themselves.
      .filter((file) => !file.startsWith(owningDir + sep))
      .filter((file) => readFileSync(file, 'utf8').includes(`"${pkg}"`))
      .map((file) => relative(repoRoot, file));

    expect(
      offenders,
      `These non-test Go files import ${pkg}, which can mint a token for any ` +
        `principal. Move the call into a _test.go file, or if a shipped code ` +
        `path genuinely needs it, that is a design change rather than an ` +
        `import.\n${offenders.join('\n')}`,
    ).toEqual([]);
  });
});
