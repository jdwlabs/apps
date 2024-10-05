import { setRemoteDefinitions } from '@nx/angular/mf';

import fallbackDefinitions from './assets/module-federation.manifest.json';
import config from './config.json';

fetch(`${config.SERVICE_DISCOVERY_BASE_URL}/config`)
  .then((res) => {
    if (!res.ok) {
      throw new Error(`Error fetching config: ${res.statusText}`);
    }
    return res.json();
  })
  .then((definitions) => setRemoteDefinitions(definitions))
  .catch((err) => {
    console.error('Fetch failed, applying fallback config:', err);
    setRemoteDefinitions(fallbackDefinitions);
  })
  .then(() => import('./bootstrap').catch((err) => console.error(err)));
