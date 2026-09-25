import { inject, Injectable, InjectionToken } from '@angular/core';
import { Route, Router } from '@angular/router';
import { MicroFrontendService } from '../micro-frontend/micro-frontend.service';
import { loadRemoteModule, setRemoteDefinitions } from '@nx/angular/mf';
import { MicroFrontendRoute } from '@jdw/frontend-container-util';
/* eslint-disable @nx/enforce-module-boundaries */
import { FallbackComponent } from '@jdw/frontend-shared-ui';
/* eslint-enable @nx/enforce-module-boundaries */

/**
 * The module-federation runtime, as a token rather than a direct import so a
 * test can hand the service a fake: the unit-test builder bundles each spec,
 * which leaves vi.mock no module boundary to replace.
 */
export const REMOTE_MODULE_FEDERATION = new InjectionToken<{
  loadRemoteModule: typeof loadRemoteModule;
  setRemoteDefinitions: typeof setRemoteDefinitions;
}>('REMOTE_MODULE_FEDERATION', {
  providedIn: 'root',
  factory: () => ({ loadRemoteModule, setRemoteDefinitions }),
});

function isUsableRoute(route: unknown): route is MicroFrontendRoute {
  if (typeof route !== 'object' || route === null) {
    return false;
  }
  const { path, remoteName, moduleName, url } = route as Record<
    string,
    unknown
  >;
  return [path, remoteName, moduleName, url].every(
    (field) => typeof field === 'string' && field.trim().length > 0,
  );
}

@Injectable({
  providedIn: 'root',
})
export class DynamicRouteLoaderService {
  private router: Router = inject(Router);
  private mfService: MicroFrontendService = inject(MicroFrontendService);
  private federation = inject(REMOTE_MODULE_FEDERATION);

  loadRoutes(): Promise<void> {
    return new Promise((resolve) => {
      this.mfService.getRoutes().subscribe({
        next: (payload) => {
          try {
            this.applyRoutes(payload);
          } finally {
            resolve();
          }
        },
        // Bootstrap is gated on this promise, so every path must settle it.
        error: (err) => {
          console.error('Failed to retrieve micro-frontend routes', err);
          resolve();
        },
      });
    });
  }

  private applyRoutes(payload: unknown): void {
    const candidates = Array.isArray(payload) ? payload : [];
    if (!Array.isArray(payload)) {
      console.error(
        'Micro-frontend route payload was not an array, ignoring it',
      );
    }

    const usable = candidates.filter(isUsableRoute);
    const discarded = candidates.length - usable.length;
    if (discarded > 0) {
      console.error(`Ignoring ${discarded} unusable micro-frontend route(s)`);
    }

    if (usable.length === 0) {
      this.handleNoUsableRoutes();
      return;
    }

    const definitions: Record<string, string> = {};
    const dynamicRoutes: Route[] = usable.map((route) => {
      definitions[route.remoteName] = route.url;
      return {
        path: route.path,
        loadChildren: () =>
          this.federation
            .loadRemoteModule(route.remoteName, route.moduleName)
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

    this.federation.setRemoteDefinitions(definitions);
    this.router.resetConfig([...this.router.config, ...dynamicRoutes]);
  }

  /**
   * Service discovery returned nothing usable. Decides what the shell shows and
   * whether the bootstrap-time remote definitions survive.
   */
  private handleNoUsableRoutes(): void {
    console.error('No usable micro-frontend routes, serving fallback');
    this.router.resetConfig([
      ...this.router.config,
      { path: '**', component: FallbackComponent },
    ]);
  }
}
