import { Route } from '@angular/router';
import { angularUsersuiFeatureCoreRoutes } from '@jdw/angular-usersui-feature-core';

export const remoteRoutes: Route[] = [
  { path: '', children: angularUsersuiFeatureCoreRoutes },
];
