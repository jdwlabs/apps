import { Environment } from '@jdw/angular-container-util';

export const environment: Environment = {
  ENVIRONMENT: process.env['NX_ENVIRONMENT'] || 'default',
};
