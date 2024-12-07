import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
} from '@angular/core';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideHttpClient } from '@angular/common/http';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import config from '../config.json';
import { appRoutes } from './app.routes';
import { provideRouter } from '@angular/router';
import { DynamicRouteLoaderService } from '@jdw/angular-container-data-access';

export const appConfig: ApplicationConfig = {
  providers: [
    provideHttpClient(),
    provideAnimations(),
    provideRouter(appRoutes),
    {
      provide: ENVIRONMENT,
      useValue: config,
    },
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
