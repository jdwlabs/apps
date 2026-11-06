import { federationOptions } from '../../../tools/module-federation/webpack';
import {
  commonShared,
  remoteShared,
  sharedModules,
} from '../../../tools/module-federation/shared';

export default federationOptions({
  name: 'usersui',
  exposes: {
    './Routes': 'apps/frontend/usersui/src/app/remote-entry/entry.routes.ts',
  },
  shared: sharedModules(commonShared, remoteShared, [
    '@angular/material-moment-adapter',
    '@angular/material/chips',
    '@angular/material/core',
    '@angular/material/datepicker',
    '@jdw/frontend-usersui-data-access',
    '@jdw/frontend-usersui-feature-core',
    '@jdw/frontend-usersui-util',
    'ag-grid-angular',
  ]),
});
