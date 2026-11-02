import { existsSync, readFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';

const root = resolve(__dirname, '../..');

function readJson<T>(path: string): T {
  return JSON.parse(readFileSync(path, 'utf8')) as T;
}

interface PackageJson {
  version?: string;
  dependencies?: Record<string, string>;
  devDependencies?: Record<string, string>;
}

interface SharedEntry {
  singleton: true;
  strictVersion?: true;
  requiredVersion: string;
  version?: string;
}

const rootPackage = readJson<PackageJson>(join(root, 'package.json'));
const npmVersions = {
  ...rootPackage.devDependencies,
  ...rootPackage.dependencies,
};

const libPaths = readJson<{
  compilerOptions: { paths: Record<string, string[]> };
}>(join(root, 'tsconfig.base.json')).compilerOptions.paths;

/**
 * Subpaths are shared as their own modules, so the version lives on the package
 * that owns them: `@angular/core/primitives/signals` -> `@angular/core`.
 */
function packageName(specifier: string): string {
  const [first, second] = specifier.split('/');
  return specifier.startsWith('@') ? `${first}/${second}` : first;
}

function npmEntry(specifier: string): SharedEntry {
  const requiredVersion = npmVersions[packageName(specifier)];
  if (!requiredVersion) {
    throw new Error(
      `Shared module ${specifier} has no version in the root package.json`,
    );
  }
  return { singleton: true, strictVersion: true, requiredVersion };
}

// A workspace library has no installed package for the plugin to read, so the
// version is given explicitly. Libraries that carry a package.json keep its
// version; the rest share under the placeholder every consumer agrees on.
function libEntry(alias: string): SharedEntry {
  const [entry] = libPaths[alias] ?? [];
  if (!entry) {
    throw new Error(`Shared library ${alias} is not in tsconfig.base.json`);
  }
  const manifest = join(dirname(dirname(resolve(root, entry))), 'package.json');
  const version = existsSync(manifest)
    ? readJson<PackageJson>(manifest).version
    : undefined;
  return version
    ? { singleton: true, requiredVersion: version, version }
    : { singleton: true, requiredVersion: '^0.0.0', version: '0.0.0' };
}

/**
 * The shared map, written out per app: sharing a module the app never imports
 * still makes the plugin build it as an extra chunk, so each app lists only
 * what its production build shared before the move off Nx Module Federation.
 */
export function sharedModules(
  ...groups: readonly (readonly string[])[]
): Record<string, SharedEntry> {
  return Object.fromEntries(
    groups
      .flat()
      .map((name) => [
        name,
        name.startsWith('@jdw/') ? libEntry(name) : npmEntry(name),
      ]),
  );
}

export const commonShared = [
  '@angular/animations/browser',
  '@angular/cdk/a11y',
  '@angular/cdk/bidi',
  '@angular/cdk/coercion',
  '@angular/cdk/collections',
  '@angular/cdk/dialog',
  '@angular/cdk/keycodes',
  '@angular/cdk/layout',
  '@angular/cdk/observers',
  '@angular/cdk/overlay',
  '@angular/cdk/platform',
  '@angular/cdk/portal',
  '@angular/cdk/private',
  '@angular/cdk/scrolling',
  '@angular/common',
  '@angular/common/http',
  '@angular/core',
  '@angular/core/primitives/di',
  '@angular/core/primitives/signals',
  '@angular/material/button',
  '@angular/material/card',
  '@angular/material/dialog',
  '@angular/material/icon',
  '@angular/material/list',
  '@angular/material/menu',
  '@angular/material/radio',
  '@angular/material/sidenav',
  '@angular/material/snack-bar',
  '@angular/material/toolbar',
  '@angular/material/tooltip',
  '@angular/platform-browser',
  '@angular/platform-browser/animations',
  '@angular/router',
  '@jdw/frontend-shared-data-access',
  '@jdw/frontend-shared-ui',
  '@jdw/frontend-shared-util',
  '@material/material-color-utilities',
  'ag-grid-community',
  'jwt-decode',
  'ngx-cookie-service',
  'rxjs',
  'rxjs/operators',
  'tslib',
] as const;

export const remoteShared = [
  '@angular/cdk/observers/private',
  '@angular/cdk/text-field',
  '@angular/forms',
  '@angular/forms/signals',
  '@angular/material/form-field',
  '@angular/material/input',
] as const;
