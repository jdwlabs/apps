import { TestBed } from '@angular/core/testing';
import { DynamicRouteLoaderService } from './dynamic-route-loader.service';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/frontend-shared-util';
import { MicroFrontendService } from '../micro-frontend/micro-frontend.service';
import { Router, Route } from '@angular/router';
import { of, throwError } from 'rxjs';
import { loadRemoteModule, setRemoteDefinitions } from '@nx/angular/mf';
/* eslint-disable @nx/enforce-module-boundaries */
import { FallbackComponent } from '@jdw/frontend-shared-ui';
/* eslint-enable @nx/enforce-module-boundaries */

vi.mock('@nx/angular/mf', () => ({
  loadRemoteModule: vi.fn(),
  setRemoteDefinitions: vi.fn(),
}));

const mockEnvironment = {
  SERVICE_DISCOVERY_BASE_URL: 'http://localhost:9000',
};

describe('DynamicRouteLoaderService', () => {
  let service: DynamicRouteLoaderService;
  let mockRouter: Router;
  let mockMfService: MicroFrontendService;

  beforeEach(() => {
    vi.clearAllMocks();

    mockRouter = {
      resetConfig: vi.fn(),
      config: [],
    } as unknown as Router;

    mockMfService = {
      getRoutes: vi.fn().mockReturnValue(of([])),
    } as unknown as MicroFrontendService;

    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: mockRouter },
        { provide: ENVIRONMENT, useValue: mockEnvironment },
        { provide: MicroFrontendService, useValue: mockMfService },
      ],
    });
    service = TestBed.inject(DynamicRouteLoaderService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should retrieve routes from MicroFrontendService', async () => {
    await service.loadRoutes();
    expect(mockMfService.getRoutes).toHaveBeenCalled();
  });

  it('should call setRemoteDefinitions with the correct definitions', async () => {
    const routes = [
      {
        path: 'example',
        remoteName: 'exampleRemote',
        moduleName: 'ExampleModule',
        url: 'http://example.com',
      },
    ];
    (mockMfService.getRoutes as vi.Mock).mockReturnValue(of(routes));

    await service.loadRoutes();

    expect(setRemoteDefinitions).toHaveBeenCalledWith({
      exampleRemote: 'http://example.com',
    });
  });

  it('should call router.resetConfig with dynamic routes', async () => {
    const routes = [
      {
        path: 'example',
        remoteName: 'exampleRemote',
        moduleName: 'ExampleModule',
        url: 'http://example.com',
      },
    ];
    (mockMfService.getRoutes as vi.Mock).mockReturnValue(of(routes));
    (loadRemoteModule as vi.Mock).mockResolvedValue({ remoteRoutes: [] });

    await service.loadRoutes();

    const expectedRoutes: Route[] = [
      ...mockRouter.config,
      { path: 'example', loadChildren: expect.any(Function) },
      { path: '**', redirectTo: '' },
    ];

    expect(mockRouter.resetConfig).toHaveBeenCalledWith(expectedRoutes);
  });

  it('should handle errors when loadRemoteModule fails', async () => {
    const routes = [
      {
        path: 'example',
        remoteName: 'exampleRemote',
        moduleName: 'ExampleModule',
        url: 'http://example.com',
      },
    ];
    (mockMfService.getRoutes as vi.Mock).mockReturnValue(of(routes));
    (loadRemoteModule as vi.Mock).mockRejectedValue(
      new Error('Failed to load remote module'),
    );

    await service.loadRoutes();

    const configCallArgs = (mockRouter.resetConfig as vi.Mock).mock.calls[0][0];
    const exampleRoute = configCallArgs.find(
      (route: any) => route.path === 'example',
    );

    const fallbackRoute = await exampleRoute.loadChildren();
    expect(fallbackRoute).toEqual([
      { path: '**', component: FallbackComponent },
    ]);
  });

  describe('unusable route payloads', () => {
    const settles = (promise: Promise<void>) =>
      Promise.race([
        promise.then(() => 'settled'),
        new Promise((resolve) => setTimeout(() => resolve('pending'), 250)),
      ]);

    it.each([
      ['null', null],
      ['undefined', undefined],
      ['an object', { remotes: {} }],
      ['a string', 'authui'],
    ])(
      'settles and serves fallback when the payload is %s',
      async (_label, payload) => {
        (mockMfService.getRoutes as vi.Mock).mockReturnValue(of(payload));

        await expect(settles(service.loadRoutes())).resolves.toBe('settled');

        expect(setRemoteDefinitions).not.toHaveBeenCalled();
        expect(mockRouter.resetConfig).toHaveBeenCalledWith([
          { path: '**', component: FallbackComponent },
        ]);
      },
    );

    it('settles when getRoutes errors instead of emitting', async () => {
      (mockMfService.getRoutes as vi.Mock).mockReturnValue(
        throwError(() => new Error('service discovery unreachable')),
      );

      await expect(settles(service.loadRoutes())).resolves.toBe('settled');
      expect(setRemoteDefinitions).not.toHaveBeenCalled();
    });

    it('preserves bootstrap definitions when the route list is empty', async () => {
      (mockMfService.getRoutes as vi.Mock).mockReturnValue(of([]));

      await service.loadRoutes();

      expect(setRemoteDefinitions).not.toHaveBeenCalled();
    });

    it('drops individual malformed routes and keeps the usable ones', async () => {
      (mockMfService.getRoutes as vi.Mock).mockReturnValue(
        of([
          {
            path: 'example',
            remoteName: 'exampleRemote',
            moduleName: 'ExampleModule',
            url: 'http://example.com',
          },
          {
            path: 'broken',
            remoteName: null,
            moduleName: 'M',
            url: 'http://b',
          },
          { path: '  ', remoteName: 'blank', moduleName: 'M', url: 'http://c' },
          null,
        ]),
      );

      await service.loadRoutes();

      expect(setRemoteDefinitions).toHaveBeenCalledWith({
        exampleRemote: 'http://example.com',
      });
      const config = (mockRouter.resetConfig as vi.Mock).mock.calls[0][0];
      expect(config.map((route: Route) => route.path)).toEqual([
        'example',
        '**',
      ]);
    });
  });
});
