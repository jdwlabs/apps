import { Route } from '@angular/router';
import { angularAuthuiFeatureCoreRoutes } from '@jdw/frontend-authui-feature-core';

export const remoteRoutes: Route[] = [
  { path: '', children: angularAuthuiFeatureCoreRoutes },
];
