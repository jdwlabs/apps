import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
} from '@angular/core';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideHttpClient, withXhr } from '@angular/common/http';
import { ENVIRONMENT, ThemeService } from '@jdw/frontend-shared-util';
import config from '../config.json';
import { appRoutes } from './app.routes';
import { provideRouter } from '@angular/router';
import { DynamicRouteLoaderService } from '@jdw/frontend-container-data-access';
import { MAT_DIALOG_DEFAULT_OPTIONS } from '@angular/material/dialog';

export const appConfig: ApplicationConfig = {
  providers: [
    provideHttpClient(withXhr()),
    provideAnimations(),
    provideRouter(appRoutes),
    {
      provide: ENVIRONMENT,
      useValue: config,
    },
    { provide: MAT_DIALOG_DEFAULT_OPTIONS, useValue: { hasBackdrop: true } },
    // Constructing the service is what applies the stored theme: it is
    // `providedIn: 'root'`, so until something injects it nothing writes
    // `data-theme` and the document keeps whatever index.html declared. The
    // switcher dialog is created lazily on first open, so hanging the boot on
    // it meant a saved theme was silently discarded on every reload.
    provideAppInitializer(() => {
      inject(ThemeService);
    }),
    provideAppInitializer(() => {
      const initializerFn = ((
        dynamicRouteLoaderService: DynamicRouteLoaderService,
      ) => {
        return () => dynamicRouteLoaderService.loadRoutes();
      })(inject(DynamicRouteLoaderService));
      return initializerFn();
    }),
  ],
};
