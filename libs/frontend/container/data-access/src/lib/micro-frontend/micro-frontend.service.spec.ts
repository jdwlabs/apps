import { TestBed } from '@angular/core/testing';

import { MicroFrontendService } from './micro-frontend.service';
import { MicroFrontendRoute } from '@jdw/frontend-container-util';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { SnackbarService } from '@jdw/frontend-shared-data-access';
import { ENVIRONMENT, getErrorMessage } from '@jdw/frontend-shared-util';

const mockSnackbarService = {
  error: vi.fn(),
};

const mockEnvironment = {
  SERVICE_DISCOVERY_BASE_URL: 'http://localhost:9000',
};

const mockRoutes: MicroFrontendRoute[] = [
  {
    path: 'home',
    remoteName: 'home',
    moduleName: 'HomeModule',
    url: '',
    icon: '',
    title: 'Home',
    description: 'Home page',
  },
  {
    path: 'about',
    remoteName: 'about',
    moduleName: 'AboutModule',
    url: '',
    icon: '',
    title: 'About',
    description: 'About page',
  },
];

vi.mock('@jdw/frontend-shared-util', async () => ({
  ...(await vi.importActual('@jdw/frontend-shared-util')),
  getErrorMessage: vi.fn(),
}));

describe('MicroFrontendService', () => {
  let service: MicroFrontendService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: SnackbarService, useValue: mockSnackbarService },
        { provide: ENVIRONMENT, useValue: mockEnvironment },
      ],
    });

    service = TestBed.inject(MicroFrontendService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('getRoutes', () => {
    it('should make an HTTP GET request and return an array of routes', () => {
      let emitted: MicroFrontendRoute[] | undefined;
      service.getRoutes().subscribe((routes) => {
        emitted = routes;
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.SERVICE_DISCOVERY_BASE_URL}/api/micro-frontends`,
      );
      expect(req.request.method).toBe('GET');
      req.flush(mockRoutes);

      expect(emitted).toEqual(mockRoutes);
    });

    it('should call handleError and return an empty array on error', () => {
      const errorResponse = new HttpErrorResponse({
        error: 'Error message',
        status: 500,
        statusText: 'Server Error',
        url: `${mockEnvironment.SERVICE_DISCOVERY_BASE_URL}/api/micro-frontends`,
      });
      vi.spyOn(service, 'handleError');

      let emitted: MicroFrontendRoute[] | undefined;
      service.getRoutes().subscribe((routes) => {
        emitted = routes;
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.SERVICE_DISCOVERY_BASE_URL}/api/micro-frontends`,
      );
      req.flush('Error message', { status: 500, statusText: 'Server Error' });

      expect(emitted).toEqual([]);
      expect(service.handleError).toHaveBeenCalledWith(errorResponse);
    });
  });

  describe('handleError', () => {
    it('should call snackbarService.error with the correct arguments and return an empty array', () => {
      const errorResponse = new HttpErrorResponse({
        error: 'Error message',
        status: 500,
      });
      const mockErrorMessage = 'Mock error message';
      vi.mocked(getErrorMessage).mockReturnValue(mockErrorMessage);

      const result = service.handleError(errorResponse);

      expect(getErrorMessage).toHaveBeenCalledWith(errorResponse);
      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        mockErrorMessage,
        { variant: 'filled', autoClose: false },
        true,
      );

      let emitted: unknown[] | undefined;
      result.subscribe((data) => {
        emitted = data;
      });
      expect(emitted).toEqual([]);
    });
  });
});
