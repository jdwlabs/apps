import { Route } from '@angular/router';
import { angularContainerFeatureCoreRoutes } from '@jdw/angular-container-feature-core';
import { loadRemoteModule } from '@nx/angular/mf';
import { FallbackComponent } from '@jdw/angular-container-feature-core';

export const appRoutes: Route[] = [
  {
    path: 'users',
    loadChildren: () =>
      loadRemoteModule('usersui', './Routes')
        .then((m) => m.remoteRoutes)
        .catch(() => [{ path: '**', component: FallbackComponent }]),
  },
  {
    path: 'auth',
    loadChildren: () =>
      loadRemoteModule('authui', './Routes')
        .then((m) => m.remoteRoutes)
        .catch(() => [{ path: '**', component: FallbackComponent }]),
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
