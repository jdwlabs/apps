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
} from '@jdw/frontend-shared-util';
import { CookieService } from 'ngx-cookie-service';
import { firstValueFrom } from 'rxjs';
import { SnackbarService } from '../snackbar/snackbar.service';

const mockSnackbarService = {
  success: vi.fn(),
  error: vi.fn(),
};

const mockCookieService = {
  get: vi.fn(),
  set: vi.fn(),
  delete: vi.fn(),
};

// Built rather than pasted: a literal JWT is a high-entropy string that reads
// as a real credential to secret scanners, and the claims are the point here.
const segment = (value: object) =>
  btoa(JSON.stringify(value)).replace(/=+$/, '');
const unsignedToken = (claims: object) =>
  `${segment({ alg: 'none', typ: 'JWT' })}.${segment(claims)}.`;

const VALID_TOKEN = unsignedToken({ user_id: 42, exp: 2_000_000_000 }); // 2033
const EXPIRED_TOKEN = unsignedToken({ user_id: 42, exp: 1_000_000_000 }); // 2001
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

  describe('restored token helpers', () => {
    // The service seeds its token stream during construction, so tests that
    // care about the seed must set the cookie before the service is built.
    const buildWith = (cookie: string | undefined) => {
      TestBed.resetTestingModule();
      mockCookieService.get.mockReturnValue(cookie ?? '');
      TestBed.configureTestingModule({
        providers: [
          provideHttpClient(),
          provideHttpClientTesting(),
          { provide: ENVIRONMENT, useValue: environmentMock },
          { provide: SnackbarService, useValue: mockSnackbarService },
          { provide: CookieService, useValue: mockCookieService },
        ],
      });
      httpTesting = TestBed.inject(HttpTestingController);
      return TestBed.inject(AuthService);
    };

    beforeEach(() => {
      mockCookieService.delete.mockClear();
      mockSnackbarService.success.mockClear();
    });

    it('decodes a valid token', () => {
      const decoded = buildWith('').getDecodedToken(VALID_TOKEN);
      expect(decoded?.user_id).toBe(42);
    });

    it('returns null rather than throwing on a malformed token', () => {
      expect(buildWith('').getDecodedToken('not-a-jwt')).toBeNull();
    });

    it('returns null when there is no token at all', () => {
      expect(buildWith('').getDecodedToken()).toBeNull();
    });

    it('treats a malformed token as expired', () => {
      expect(buildWith('').isTokenExpired('not-a-jwt')).toBe(true);
    });

    it('treats a past exp as expired', () => {
      expect(buildWith('').isTokenExpired(EXPIRED_TOKEN)).toBe(true);
    });

    it('treats a future exp as live', () => {
      expect(buildWith('').isTokenExpired(VALID_TOKEN)).toBe(false);
    });

    it('reads the user id from the token', () => {
      expect(buildWith('').getUserIdFromToken(VALID_TOKEN)).toBe(42);
    });

    it('returns an empty user id when the token cannot be decoded', () => {
      expect(buildWith('').getUserIdFromToken('not-a-jwt')).toBe('');
    });

    it('seeds token$ from the cookie', async () => {
      const service = buildWith(VALID_TOKEN);
      await expect(firstValueFrom(service.token$)).resolves.toBe(VALID_TOKEN);
    });

    it('emits null on token$ when no cookie is present', async () => {
      const service = buildWith('');
      await expect(firstValueFrom(service.token$)).resolves.toBeNull();
    });

    it('deletes both cookie forms and emits null on sign out', async () => {
      const service = buildWith(VALID_TOKEN);

      service.signOut(false);

      expect(mockCookieService.delete).toHaveBeenCalledWith('jwtToken');
      expect(mockCookieService.delete).toHaveBeenCalledWith('jwtToken', '/');
      await expect(firstValueFrom(service.token$)).resolves.toBeNull();
    });

    it('shows a snackbar on sign out by default', () => {
      buildWith(VALID_TOKEN).signOut();

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        'Sign out successful',
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('stays silent when sign out is told not to announce itself', () => {
      buildWith(VALID_TOKEN).signOut(false);

      expect(mockSnackbarService.success).not.toHaveBeenCalled();
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
