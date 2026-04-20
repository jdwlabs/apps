import { Route } from '@angular/router';
import { angularRolesuiFeatureCoreRoutes } from '@jdw/angular-rolesui-feature-core';

export const remoteRoutes: Route[] = [
  { path: '', children: angularRolesuiFeatureCoreRoutes },
];
