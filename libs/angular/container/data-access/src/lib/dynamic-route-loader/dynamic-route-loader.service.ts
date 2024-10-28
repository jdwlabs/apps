import { inject, Injectable } from '@angular/core';
import { Route, Router } from '@angular/router';
import { MicroFrontendService } from '../micro-frontend/micro-frontend.service';
import { loadRemoteModule, setRemoteDefinitions } from '@nx/angular/mf';
import { FallbackComponent } from '@jdw/angular-shared-ui';

@Injectable({
  providedIn: 'root',
})
export class DynamicRouteLoaderService {
  private router: Router = inject(Router);
  private mfService: MicroFrontendService = inject(MicroFrontendService);

  loadRoutes(): Promise<void> {
    return new Promise((resolve) => {
      this.mfService.getRoutes().subscribe((routes) => {
        const definitions: Record<string, string> = {};
        const dynamicRoutes: Route[] = routes.map((route) => {
          definitions[route.remoteName] = route.url;
          return {
            path: route.path,
            loadChildren: () =>
              loadRemoteModule(route.remoteName, route.moduleName)
                .then((m) => m.remoteRoutes)
                .catch((err) => {
                  console.error('Failed to load remote', err);
                  return [{ path: '**', component: FallbackComponent }];
                }),
          };
        });
        dynamicRoutes.push({
          path: '**',
          redirectTo: '',
        });
        setRemoteDefinitions(definitions);
        this.router.resetConfig([...this.router.config, ...dynamicRoutes]);
        resolve();
      });
    });
  }
}
