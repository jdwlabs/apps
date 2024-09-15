import { Route } from '@angular/router';
import { angularAuthuiFeatureCoreRoutes } from '@jdw/angular-authui-feature-core';

export const remoteRoutes: Route[] = [
  { path: '', children: angularAuthuiFeatureCoreRoutes },
];
