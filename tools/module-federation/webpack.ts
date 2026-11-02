import { ModuleFederationPlugin } from '@module-federation/enhanced/webpack';
import { readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';

type FederationOptions = ConstructorParameters<
  typeof ModuleFederationPlugin
>[0];

const root = resolve(__dirname, '../..');

const libPaths: Record<string, string[]> = JSON.parse(
  readFileSync(join(root, 'tsconfig.base.json'), 'utf8'),
).compilerOptions.paths;

// The plugin resolves exposed and shared modules outside the Angular
// compilation, where tsconfig paths do not apply. Only workspace libraries are
// aliased: a `<remote>/Routes` path here would bundle that remote into the host.
const libAliases = Object.fromEntries(
  Object.entries(libPaths)
    .filter(([alias]) => alias.startsWith('@jdw/'))
    .map(([alias, [entry]]) => [alias, join(root, entry)]),
);

export const remoteEntryFilename = 'remoteEntry.mjs';

export function federationOptions(
  options: Pick<FederationOptions, 'name' | 'exposes' | 'shared'> & {
    shared: NonNullable<FederationOptions['shared']>;
  },
): FederationOptions {
  return {
    ...options,
    filename: remoteEntryFilename,
    library: { type: 'module' },
    dts: false,
    // Remotes are registered at runtime from service discovery.
    remotes: {},
  };
}

export function withFederation(options: FederationOptions) {
  return (config: any) => ({
    ...config,
    output: {
      ...config.output,
      uniqueName: options.name,
      publicPath: 'auto',
    },
    experiments: { ...config.experiments, outputModule: true },
    resolve: {
      ...config.resolve,
      modules: [...(config.resolve?.modules ?? ['node_modules']), root],
      alias: { ...config.resolve?.alias, ...libAliases },
    },
    optimization: { ...config.optimization, runtimeChunk: false },
    plugins: [...(config.plugins ?? []), new ModuleFederationPlugin(options)],
  });
}
