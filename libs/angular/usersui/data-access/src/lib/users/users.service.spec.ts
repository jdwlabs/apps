import { TestBed } from '@angular/core/testing';

import { UsersService } from './users.service';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { AuthService, SnackbarService } from '@jdw/angular-shared-data-access';
import { User } from '@jdw/angular-usersui-util';
import { EMPTY } from 'rxjs';

const mockAuthService = {
  getToken: jest.fn(),
};

const mockSnackbarService = {
  error: jest.fn(),
};

const environmentMock = {
  AUTH_BASE_URL: 'http://localhost:8080',
};

describe('UsersService', () => {
  let service: UsersService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: AuthService,
          useValue: mockAuthService,
        },
        {
          provide: SnackbarService,
          useValue: mockSnackbarService,
        },
        {
          provide: ENVIRONMENT,
          useValue: environmentMock,
        },
      ],
    });
    service = TestBed.inject(UsersService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('getUsers', () => {
    it('should send a GET request with the correct headers and return users', () => {
      const mockUsers: User[] = [
        {
          id: 1,
          emailAddress: 'user1@jdwkube.com',
          password: 'P@ssw0rd',
          status: 'ACTIVE',
          roles: [],
          profile: null,
          createdByUserId: 1,
          createdTime: '2024-08-09T01:02:34.567+00:00',
          modifiedByUserId: 1,
          modifiedTime: '2024-08-09T01:02:34.567+00:00',
        },
        {
          id: 2,
          emailAddress: 'user2@jdwkube.com',
          password: 'P@ssw0rd',
          status: 'ACTIVE',
          roles: [],
          profile: null,
          createdByUserId: 1,
          createdTime: '2024-08-09T01:02:34.567+00:00',
          modifiedByUserId: 1,
          modifiedTime: '2024-08-09T01:02:34.567+00:00',
        },
      ];
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.getUsers().subscribe((users) => {
        expect(users).toEqual(mockUsers);
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(mockUsers);
    });

    it('should handle error when the request fails and shows the correct error message', () => {
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.getUsers().subscribe({
        next: () => fail('Expected an error, but got a success response'),
        error: (error) => {
          expect(error).toBe(EMPTY);
        },
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users`,
      );
      req.flush(
        { message: 'Error' },
        { status: 500, statusText: 'Internal Server Error' },
      );

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'An unexpected error occurred on our server. Please try again later.',
        {
          variant: 'filled',
          autoClose: false,
        },
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
        {
          variant: 'filled',
          autoClose: false,
        },
        true,
      );
    });
  });
});
