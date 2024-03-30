import { Environment } from '@jdw/angular-container-util';

export const environment: Environment = {
  ENVIRONMENT: process.env['ENVIRONMENT'] || 'default',
};
