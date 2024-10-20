import { TestBed } from '@angular/core/testing';

import { RolesService } from './roles.service';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { ENVIRONMENT, Role } from '@jdw/angular-shared-util';
import { EMPTY } from 'rxjs';
import { AuthService } from '../auth/auth.service';
import { SnackbarService } from '../snackbar/snackbar.service';

const mockAuthService = {
  getToken: jest.fn(),
};

const mockSnackbarService = {
  error: jest.fn(),
};

const mockEnvironment = {
  AUTH_BASE_URL: 'http://localhost:8080',
};

describe('RolesService', () => {
  let service: RolesService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: mockAuthService },
        { provide: SnackbarService, useValue: mockSnackbarService },
        { provide: ENVIRONMENT, useValue: mockEnvironment },
      ],
    });

    service = TestBed.inject(RolesService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('getRoles', () => {
    it('should send a GET request with the correct headers and return roles', () => {
      const mockRoles: Role[] = [
        {
          id: 1,
          name: 'Admin',
          description: 'Administrator role with full access',
          status: 'ACTIVE',
          users: [],
          createdByUserId: 1,
          createdTime: '2024-08-09T01:02:34.567+00:00',
          modifiedByUserId: 2,
          modifiedTime: '2024-09-01T10:11:12.000+00:00',
        },
        {
          id: 2,
          name: 'User',
          description: 'Regular user with limited access',
          status: 'ACTIVE',
          users: [],
          createdByUserId: 1,
          createdTime: '2024-08-10T01:02:34.567+00:00',
          modifiedByUserId: 2,
          modifiedTime: '2024-09-02T10:11:12.000+00:00',
        },
      ];
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.getRoles().subscribe((roles) => {
        expect(roles).toEqual(mockRoles);
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      req.flush(mockRoles);
    });

    it('should handle error when the request fails and show the correct error message', () => {
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.getRoles().subscribe({
        next: () => fail('Expected an error, but got a success response'),
        error: (error) => {
          expect(error).toBe(EMPTY);
        },
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles`,
      );
      req.flush(
        { message: 'Error' },
        { status: 500, statusText: 'Internal Server Error' },
      );

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'An unexpected error occurred on our server. Please try again later.',
        { variant: 'filled', autoClose: false },
        true,
      );
    });
  });

  describe('getRole', () => {
    it('should send a GET request with the correct headers and return a role by ID', () => {
      const mockRole: Role = {
        id: 1,
        name: 'Admin',
        description: 'Administrator role with full access',
        status: 'ACTIVE',
        users: [
          {
            userId: 1,
            roleId: 1,
            createdByUserId: 1,
            createdTime: '2024-09-01T10:11:12.000+00:00',
          },
        ],
        createdByUserId: 1,
        createdTime: '2024-08-09T01:02:34.567+00:00',
        modifiedByUserId: 2,
        modifiedTime: '2024-09-01T10:11:12.000+00:00',
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.getRole('1').subscribe((role) => {
        expect(role).toEqual(mockRole);
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles/1`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      req.flush(mockRole);
    });

    it('should handle error when the request fails', () => {
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.getRole('1').subscribe({
        next: () => fail('Expected an error, but got a success response'),
        error: (error) => {
          expect(error).toBe(EMPTY);
        },
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles/1`,
      );
      req.flush({ message: 'Error' }, { status: 404, statusText: 'Not Found' });

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'An unexpected error occurred on our server. Please try again later.',
        { variant: 'filled', autoClose: false },
        true,
      );
    });
  });

  describe('handleError', () => {
    it('should show error message via snackbar on 500 error', () => {
      const mockError = new HttpErrorResponse({ status: 500 });

      service.handleError(mockError);

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'An unexpected error occurred on our server. Please try again later.',
        { variant: 'filled', autoClose: false },
        true,
      );
    });

    it('should handle different error codes correctly', () => {
      const mockError = new HttpErrorResponse({
        status: 404,
        statusText: 'Not Found',
      });

      service.handleError(mockError);

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'An unexpected error occurred on our server. Please try again later.',
        { variant: 'filled', autoClose: false },
        true,
      );
    });
  });
});
