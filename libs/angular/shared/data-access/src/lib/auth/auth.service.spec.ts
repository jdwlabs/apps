import { TestBed } from '@angular/core/testing';

import { AuthService } from './auth.service';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import {
  CreateUserRequest,
  ENVIRONMENT,
  LoginUserRequest,
} from '@jdw/angular-shared-util';
import { CookieService } from 'ngx-cookie-service';
import { SnackbarService } from '../snackbar/snackbar.service';

const mockSnackbarService = {
  success: jest.fn(),
  error: jest.fn(),
};

const mockCookieService = {
  get: jest.fn(),
  set: jest.fn(),
};
const environmentMock = {
  AUTH_BASE_URL: 'http://localhost:8080',
};

describe('AuthService', () => {
  let service: AuthService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: environmentMock,
        },
        {
          provide: SnackbarService,
          useValue: mockSnackbarService,
        },
        {
          provide: CookieService,
          useValue: mockCookieService,
        },
      ],
    });
    service = TestBed.inject(AuthService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('getToken', () => {
    it('should retrieve the JWT token from cookies', () => {
      mockCookieService.get.mockReturnValue('mockJwtToken');
      const token = service.getToken();
      expect(token).toBe('mockJwtToken');
      expect(mockCookieService.get).toHaveBeenCalledWith('jwtToken');
    });
  });

  describe('signIn', () => {
    it('should send a POST request and store JWT token on success', () => {
      const mockUser: LoginUserRequest = {
        emailAddress: 'user@jdw.com',
        password: 'P@ssw0rd',
      };
      const mockResponse = { jwtToken: 'mockJwtToken' };

      service.signIn(mockUser).subscribe();

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/auth/authenticate`,
      );
      expect(req.request.method).toBe('POST');
      req.flush(mockResponse);

      expect(mockCookieService.set).toHaveBeenCalledWith(
        'jwtToken',
        'mockJwtToken',
        {
          secure: true,
          path: '/',
        },
      );
      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        'Sign in successful',
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should handle sign-in error and show the correct error message', () => {
      const mockUser: LoginUserRequest = {
        emailAddress: 'user@jdw.com',
        password: 'P@ssw0rd',
      };

      service.signIn(mockUser).subscribe();

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/auth/authenticate`,
      );
      req.flush(
        { message: 'Unauthorized' },
        { status: 401, statusText: 'Unauthorized' },
      );

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'Invalid email or password',
        { variant: 'filled', autoClose: false },
        true,
      );
    });
  });

  describe('signUp', () => {
    it('should send a POST request and handle success', () => {
      const mockUser: CreateUserRequest = {
        emailAddress: 'user@jdw.com',
        password: 'P@ssw0rd',
      };

      service.signUp(mockUser).subscribe();

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/auth/user`,
      );
      expect(req.request.method).toBe('POST');
      req.flush({});

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        'Sign in successful',
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should handle sign-up error and show the correct error message', () => {
      const mockUser: CreateUserRequest = {
        emailAddress: 'user@jdw.com',
        password: 'P@ssw0rd',
      };

      service.signUp(mockUser).subscribe();

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/auth/user`,
      );
      req.flush(
        { message: 'Bad Request' },
        { status: 400, statusText: 'Bad Request' },
      );

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'Invalid email or password',
        { variant: 'filled', autoClose: false },
        true,
      );
    });
  });

  describe('handleSignInError', () => {
    it('should return a specific message for 401 status', () => {
      const mockError = new HttpErrorResponse({ status: 401 });

      service.handleSignInError(mockError);

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'Invalid email or password',
        { variant: 'filled', autoClose: false },
        true,
      );
    });

    it('should return a generic error message for other statuses', () => {
      const mockError = new HttpErrorResponse({ status: 500 });

      service.handleSignInError(mockError);

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'An unexpected error occurred on our server. Please try again later.',
        { variant: 'filled', autoClose: false },
        true,
      );
    });
  });
});
