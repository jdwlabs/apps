import { TestBed } from '@angular/core/testing';

import { MicroFrontendService } from './micro-frontend.service';
import { MicroFrontendRoute } from '@jdw/angular-container-util';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { SnackbarService } from '@jdw/angular-shared-data-access';
import { ENVIRONMENT, getErrorMessage } from '@jdw/angular-shared-util';

const mockSnackbarService = {
  error: jest.fn(),
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

jest.mock('@jdw/angular-shared-util', () => ({
  ...jest.requireActual('@jdw/angular-shared-util'),
  getErrorMessage: jest.fn(),
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
    it('should make an HTTP GET request and return an array of routes', (done) => {
      service.getRoutes().subscribe((routes) => {
        expect(routes).toEqual(mockRoutes);
        done();
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.SERVICE_DISCOVERY_BASE_URL}/api/micro-frontends`,
      );
      expect(req.request.method).toBe('GET');
      req.flush(mockRoutes);
    });

    it('should call handleError and return an empty array on error', (done) => {
      const errorResponse = new HttpErrorResponse({
        error: 'Error message',
        status: 500,
        statusText: 'Server Error',
        url: `${mockEnvironment.SERVICE_DISCOVERY_BASE_URL}/api/micro-frontends`,
      });
      jest.spyOn(service, 'handleError');

      service.getRoutes().subscribe((routes) => {
        expect(routes).toEqual([]);
        expect(service.handleError).toHaveBeenCalledWith(errorResponse);
        done();
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.SERVICE_DISCOVERY_BASE_URL}/api/micro-frontends`,
      );
      req.flush('Error message', { status: 500, statusText: 'Server Error' });
    });
  });

  describe('handleError', () => {
    it('should call snackbarService.error with the correct arguments and return an empty array', (done) => {
      const errorResponse = new HttpErrorResponse({
        error: 'Error message',
        status: 500,
      });
      const mockErrorMessage = 'Mock error message';
      (getErrorMessage as jest.Mock).mockReturnValue(mockErrorMessage);

      const result = service.handleError(errorResponse);

      expect(getErrorMessage).toHaveBeenCalledWith(errorResponse);
      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        mockErrorMessage,
        { variant: 'filled', autoClose: false },
        true,
      );

      result.subscribe((data) => {
        expect(data).toEqual([]);
        done();
      });
    });
  });
});
