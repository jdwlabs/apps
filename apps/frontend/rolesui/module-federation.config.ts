import { federationOptions } from '../../../tools/module-federation/webpack';
import {
  commonShared,
  remoteShared,
  sharedModules,
} from '../../../tools/module-federation/shared';

export default federationOptions({
  name: 'rolesui',
  exposes: {
    './Routes': 'apps/frontend/rolesui/src/app/remote-entry/entry.routes.ts',
  },
  shared: sharedModules(commonShared, remoteShared, [
    '@angular/material/autocomplete',
    '@jdw/frontend-rolesui-feature-core',
    '@jdw/frontend-rolesui-util',
    'ag-grid-angular',
  ]),
});
