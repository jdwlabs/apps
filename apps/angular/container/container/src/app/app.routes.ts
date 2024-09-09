import { Route } from '@angular/router';
import { angularContainerFeatureCoreRoutes } from '@jdw/angular-container-feature-core';
import { loadRemoteModule } from '@nx/angular/mf';

export const appRoutes: Route[] = [
  {
    path: 'authui',
    loadChildren: () =>
      loadRemoteModule('authui', './Routes').then((m) => m.remoteRoutes),
  },
  {
    path: '',
    children: angularContainerFeatureCoreRoutes,
  },
  {
    path: '**',
    redirectTo: '',
  },
];
