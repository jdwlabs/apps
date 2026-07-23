import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';
import { provideRouter } from '@angular/router';
import { appRoutes } from './app.routes';
import { provideHttpClient, withXhr } from '@angular/common/http';
import { provideAnimations } from '@angular/platform-browser/animations';
import config from '../config.json';
import { ENVIRONMENT } from '@jdw/frontend-shared-util';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideHttpClient(withXhr()),
    provideAnimations(),
    provideRouter(appRoutes),
    {
      provide: ENVIRONMENT,
      useValue: config,
    },
  ],
};
