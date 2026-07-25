import { setRemoteDefinitions } from '@nx/angular/mf';

import fallbackDefinitions from './module-federation.manifest.json';
import config from './config.json';
import { resolveRemoteDefinitions } from './remote-definitions';

fetch(`${config.SERVICE_DISCOVERY_BASE_URL}/api/remotes`)
  .then((res) => {
    if (!res.ok) {
      throw new Error(`Error fetching config: ${res.statusText}`);
    }
    return res.json();
  })
  .then((payload) => {
    const { definitions, usedFallback, ignored } = resolveRemoteDefinitions(
      payload,
      fallbackDefinitions,
    );

    if (ignored.length > 0) {
      console.error(
        `Ignoring unusable remote definitions: ${ignored.join(', ')}`,
      );
    }
    if (usedFallback) {
      console.error('No usable remote definitions, applying fallback config');
    }

    setRemoteDefinitions(definitions);
  })
  .catch((err) => {
    console.error('Fetch failed, applying fallback config:', err);
    setRemoteDefinitions(fallbackDefinitions);
  })
  .then(() => import('./bootstrap').catch((err) => console.error(err)));
