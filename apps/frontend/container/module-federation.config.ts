import { federationOptions } from '../../../tools/module-federation/webpack';
import {
  commonShared,
  sharedModules,
} from '../../../tools/module-federation/shared';

export default federationOptions({
  name: 'container',
  shared: sharedModules(commonShared, [
    '@jdw/frontend-container-data-access',
    '@jdw/frontend-container-feature-core',
  ]),
});
