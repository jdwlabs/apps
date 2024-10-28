import { TestBed } from '@angular/core/testing';
import { DynamicRouteLoaderService } from './dynamic-route-loader.service';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { MicroFrontendService } from '@jdw/angular-container-data-access';
import { Router, Route } from '@angular/router';
import { of } from 'rxjs';
import { FallbackComponent } from '@jdw/angular-shared-ui';
import { loadRemoteModule, setRemoteDefinitions } from '@nx/angular/mf';

jest.mock('@nx/angular/mf', () => ({
  loadRemoteModule: jest.fn(),
  setRemoteDefinitions: jest.fn(),
}));

const mockEnvironment = {
  SERVICE_DISCOVERY_BASE_URL: 'http://localhost:9000',
};

describe('DynamicRouteLoaderService', () => {
  let service: DynamicRouteLoaderService;
  let mockRouter: Router;
  let mockMfService: MicroFrontendService;

  beforeEach(() => {
    mockRouter = {
      resetConfig: jest.fn(),
      config: [],
    } as unknown as Router;

    mockMfService = {
      getRoutes: jest.fn().mockReturnValue(of([])),
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
    (mockMfService.getRoutes as jest.Mock).mockReturnValue(of(routes));

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
    (mockMfService.getRoutes as jest.Mock).mockReturnValue(of(routes));
    (loadRemoteModule as jest.Mock).mockResolvedValue({ remoteRoutes: [] });

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
    (mockMfService.getRoutes as jest.Mock).mockReturnValue(of(routes));
    (loadRemoteModule as jest.Mock).mockRejectedValue(
      new Error('Failed to load remote module'),
    );

    await service.loadRoutes();

    const configCallArgs = (mockRouter.resetConfig as jest.Mock).mock
      .calls[0][0];
    const exampleRoute = configCallArgs.find(
      (route: any) => route.path === 'example',
    );

    const fallbackRoute = await exampleRoute.loadChildren();
    expect(fallbackRoute).toEqual([
      { path: '**', component: FallbackComponent },
    ]);
  });
});
