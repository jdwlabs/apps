import '@analogjs/vitest-angular/setup-zone';
import { setupTestBed } from '@analogjs/vitest-angular/setup-testbed';

// Apps use provideZoneChangeDetection (not zoneless) in production, so tests
// must run zone-based too for behavioral parity with jest-preset-angular's
// setupZoneTestEnv.
setupTestBed({ zoneless: false });
