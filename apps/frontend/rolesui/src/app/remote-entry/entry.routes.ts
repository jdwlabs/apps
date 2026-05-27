import { Route } from '@angular/router';
import { angularRolesuiFeatureCoreRoutes } from '@jdw/frontend-rolesui-feature-core';

export const remoteRoutes: Route[] = [
  { path: '', children: angularRolesuiFeatureCoreRoutes },
];
