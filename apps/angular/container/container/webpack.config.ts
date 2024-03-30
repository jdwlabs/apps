import { withModuleFederation } from '@nx/angular/module-federation';
import config from './module-federation.config';
import webpack = require('webpack');

function getClientEnvironment() {
  // Stringify all values so we can feed into webpack DefinePlugin
  return {
    'process.env': Object.keys(process.env).reduce((env, key) => {
      env[key] = JSON.stringify(process.env[key]);
      return env;
    }, {}),
  };
}

module.exports = (config: { mode: any; plugins: webpack.DefinePlugin[] }) => {
  // Overwrite the mode set by Angular if the NODE_ENV is set
  config.mode = process.env.NODE_ENV || config.mode;
  config.plugins.push(new webpack.DefinePlugin(getClientEnvironment()));
  return config;
};

export default withModuleFederation(config);
