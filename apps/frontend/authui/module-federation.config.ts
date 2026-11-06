import { federationOptions } from '../../../tools/module-federation/webpack';
import {
  commonShared,
  remoteShared,
  sharedModules,
} from '../../../tools/module-federation/shared';

export default federationOptions({
  name: 'authui',
  exposes: {
    './Routes': 'apps/frontend/authui/src/app/remote-entry/entry.routes.ts',
  },
  shared: sharedModules(commonShared, remoteShared, [
    '@jdw/frontend-authui-feature-core',
  ]),
});
