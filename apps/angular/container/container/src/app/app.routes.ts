import { Route } from '@angular/router';
import { angularContainerFeatureCoreRoutes } from '@jdw/angular-container-feature-core';

export const appRoutes: Route[] = [
  {
    path: '',
    children: angularContainerFeatureCoreRoutes,
  },
  {
    path: '**',
    redirectTo: '',
  },
];
