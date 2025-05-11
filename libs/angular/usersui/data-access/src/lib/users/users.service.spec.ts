import { TestBed } from '@angular/core/testing';

import { UsersService } from './users.service';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import {
  HttpTestingController,
  provideHttpClientTesting,
  TestRequest,
} from '@angular/common/http/testing';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { AuthService, SnackbarService } from '@jdw/angular-shared-data-access';
import { AddUser, EditUser, User } from '@jdw/angular-usersui-util';
import { EMPTY } from 'rxjs';

const mockAuthService = {
  getToken: jest.fn(),
};

const mockSnackbarService = {
  success: jest.fn(),
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

  describe('getUser', () => {
    it('should send a GET request with the correct headers and return a user by ID', () => {
      const mockUser: User = {
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
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.getUser('1').subscribe((user) => {
        expect(user).toEqual(mockUser);
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users/1`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(mockUser);
    });

    it('should handle error when the request fails', () => {
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.getUser('1').subscribe({
        next: () => fail('Expected an error, but got a success response'),
        error: (error) => {
          expect(error).toBe(EMPTY);
        },
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users/1`,
      );
      req.flush({ message: 'Error' }, { status: 404, statusText: 'Not Found' });

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

  describe('deleteUser', () => {
    it('should send a DELETE request with the correct headers and show success message', () => {
      const userId = 1;
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.deleteUser(userId).subscribe((response) => {
        expect(response).toBeTruthy();
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users/${userId}`,
      );
      expect(req.request.method).toBe('DELETE');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush({});

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `User ${userId} Deleted Successfully`,
        {
          variant: 'filled',
          autoClose: true,
        },
        true,
      );
    });

    it('should handle error when the request fails', () => {
      const userId = 1;
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.deleteUser(userId).subscribe({
        next: () => fail('Expected an error, but got a success response'),
        error: (error) => {
          expect(error).toBeInstanceOf(HttpErrorResponse);
        },
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users/${userId}`,
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

  describe('addUser', () => {
    it('should send a POST request with the correct headers and show success message', () => {
      const mockUser: AddUser = {
        emailAddress: 'newuser@jdwkube.com',
        password: 'P@ssw0rd',
      };
      const createdUser: User = {
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
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.addUser(mockUser).subscribe((user) => {
        expect(user).toEqual(createdUser);
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users`,
      );
      expect(req.request.method).toBe('POST');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(createdUser);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        'User added Successfully',
        {
          variant: 'filled',
          autoClose: true,
        },
        true,
      );
    });

    it('should handle error when the request fails', () => {
      const mockUser: AddUser = {
        emailAddress: 'newuser@jdwkube.com',
        password: 'P@ssw0rd',
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.addUser(mockUser).subscribe({
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

  describe('editUser', () => {
    it('should send a PUT request with the correct headers and show success message', () => {
      const userId = 1;
      const mockUserUpdate: EditUser = {
        emailAddress: 'user1@jdwkube.com',
        password: 'P@ssw0rd',
      };
      const updatedUser: User = {
        id: userId,
        emailAddress: 'user1@jdwkube.com',
        password: 'P@ssw0rd',
        status: 'INACTIVE',
        roles: [],
        profile: null,
        createdByUserId: 1,
        createdTime: '2024-08-09T01:02:34.567+00:00',
        modifiedByUserId: 1,
        modifiedTime: '2024-08-09T02:00:00.000+00:00',
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.editUser(userId, mockUserUpdate).subscribe((user) => {
        expect(user).toEqual(updatedUser);
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users/${userId}`,
      );
      expect(req.request.method).toBe('PUT');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(updatedUser);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        'User edited Successfully',
        {
          variant: 'filled',
          autoClose: true,
        },
        true,
      );
    });

    it('should handle error when the request fails', () => {
      const userId = 1;
      const mockUserUpdate: EditUser = {
        emailAddress: 'user1@jdwkube.com',
        password: 'P@ssw0rd',
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.editUser(userId, mockUserUpdate).subscribe({
        next: () => fail('Expected an error, but got a success response'),
        error: (error) => {
          expect(error).toBe(EMPTY);
        },
      });

      const req = httpTesting.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/users/${userId}`,
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

  describe('assignRolesToUser', () => {
    const userId = 42;
    const grantUrl = `${environmentMock.AUTH_BASE_URL}/api/users/${userId}/roles/grant`;
    const revokeUrl = `${environmentMock.AUTH_BASE_URL}/api/users/${userId}/roles/revoke`;
    const token = 'mockJwtToken';

    beforeEach(() => {
      mockAuthService.getToken.mockReturnValue(token);
    });

    it('should call only the grant endpoint when rolesToAdd is non-empty', () => {
      const newRoles = [1, 2, 3];
      const returnedUser: User = {
        id: userId,
        emailAddress: 'u',
        password: '',
        status: 'ACTIVE',
        roles: [],
        profile: null,
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };

      service.assignRolesToUser(userId, newRoles, []).subscribe((user) => {
        expect(user).toEqual(returnedUser);
      });

      // Expect a single PUT to grant URL
      const req = httpTesting.expectOne(grantUrl);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual(newRoles);
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      // No revoke call should be made
      httpTesting.expectNone(revokeUrl);

      // Flush simulated backend response
      req.flush(returnedUser);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Granted ${newRoles.length} role(s)`,
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should call only the revoke endpoint when rolesToRemove is non-empty', () => {
      const removedRoles = [9, 8];
      const returnedUser: User = {
        id: userId,
        emailAddress: 'u',
        password: '',
        status: 'ACTIVE',
        roles: [],
        profile: null,
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };

      service.assignRolesToUser(userId, [], removedRoles).subscribe((user) => {
        expect(user).toEqual(returnedUser);
      });

      const req = httpTesting.expectOne(revokeUrl);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual(removedRoles);

      httpTesting.expectNone(grantUrl);

      req.flush(returnedUser);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Revoked ${removedRoles.length} role(s)`,
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should call both endpoints in parallel and return the revoke result', () => {
      const toAdd = [10];
      const toRemove = [20];
      const userAfterGrant: User = {
        id: userId,
        emailAddress: 'u',
        password: '',
        status: 'ACTIVE',
        roles: [],
        profile: null,
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };
      const userAfterRevoke: User = {
        id: userId,
        emailAddress: 'u2',
        password: '',
        status: 'INACTIVE',
        roles: [],
        profile: null,
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };

      service.assignRolesToUser(userId, toAdd, toRemove).subscribe((user) => {
        expect(user).toEqual(userAfterRevoke);
      });

      // match() predicate sees HttpRequest, so use req.url
      const requests: TestRequest[] = httpTesting.match(
        (req) => req.url === grantUrl || req.url === revokeUrl,
      );
      expect(requests.length).toBe(2);

      // Now each TestRequest has a `.request` containing the HttpRequest
      const grantReq = requests.find((r) => r.request.url === grantUrl)!;
      const revokeReq = requests.find((r) => r.request.url === revokeUrl)!;

      expect(grantReq.request.method).toBe('PUT');
      expect(grantReq.request.body).toEqual(toAdd);

      expect(revokeReq.request.method).toBe('PUT');
      expect(revokeReq.request.body).toEqual(toRemove);

      // Flush in sequence
      grantReq.flush(userAfterGrant);
      revokeReq.flush(userAfterRevoke);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Granted ${toAdd.length} role(s)`,
        { variant: 'filled', autoClose: true },
        true,
      );
      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Revoked ${toRemove.length} role(s)`,
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should return EMPTY when there are no role changes', () => {
      let called = false;
      service.assignRolesToUser(userId, [], []).subscribe({
        next: () => (called = true),
        complete: () => {},
      });

      // No HTTP calls should be made
      httpTesting.expectNone(grantUrl);
      httpTesting.expectNone(revokeUrl);

      expect(called).toBe(false);
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

    it('should show a custom message for 404 errors', () => {
      const mockError = new HttpErrorResponse({ status: 404 });

      service.handleError(mockError);

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'The requested resource could not be found. Please verify the URL or resource and try again.',
        {
          variant: 'filled',
          autoClose: false,
        },
        true,
      );
    });

    it('should handle forbidden error (403)', () => {
      const mockError = new HttpErrorResponse({ status: 403 });

      service.handleError(mockError);

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'You do not have the necessary permissions to perform this action.',
        {
          variant: 'filled',
          autoClose: false,
        },
        true,
      );
    });

    it('should handle network error', () => {
      const mockError = new HttpErrorResponse({
        error: new ErrorEvent('NetworkError', {
          message: 'Unable to connect to the server',
        }),
      });

      service.handleError(mockError);

      expect(mockSnackbarService.error).toHaveBeenCalledWith(
        'An error occurred. Please contact support if the issue persists.',
        {
          variant: 'filled',
          autoClose: false,
        },
        true,
      );
    });
  });
});
