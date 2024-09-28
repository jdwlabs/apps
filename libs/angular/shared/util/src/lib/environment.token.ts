import { InjectionToken } from '@angular/core';

export type Environment = {
  ENVIRONMENT: string;
  AUTH_BASE_URL: string;
} & Record<string, string>;

export const ENVIRONMENT = new InjectionToken<Environment>('ENVIRONMENT');
