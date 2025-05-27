import { TestBed } from '@angular/core/testing';

import { RolesService } from './roles.service';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { ENVIRONMENT, Role, User } from '@jdw/angular-shared-util';
import { EMPTY } from 'rxjs';
import { AuthService } from '../auth/auth.service';
import { SnackbarService } from '../snackbar/snackbar.service';

const mockAuthService = {
  getToken: jest.fn(),
};

const mockSnackbarService = {
  error: jest.fn(),
  success: jest.fn(),
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

  describe('deleteRole', () => {
    it('should send DELETE request and show success snackbar', () => {
      const roleId = 123;
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.deleteRole(roleId).subscribe((res) => {
        expect(res).toEqual({});
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles/${roleId}`,
      );
      expect(req.request.method).toBe('DELETE');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      req.flush({});

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Role ${roleId} Deleted Successfully`,
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should handle error on DELETE failure', () => {
      const roleId = 123;
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.deleteRole(roleId).subscribe({
        next: () => fail('Expected error'),
        error: (e) => {
          expect(e).toBe(EMPTY);
        },
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles/${roleId}`,
      );
      req.flush({}, { status: 500, statusText: 'Internal Server Error' });

      expect(mockSnackbarService.error).toHaveBeenCalled();
    });
  });

  describe('addRole', () => {
    it('should send POST request and show success snackbar', () => {
      const newRole = {
        name: 'NewRole',
        description: 'desc',
        status: 'ACTIVE',
      };
      const responseRole: Role = {
        ...newRole,
        id: 5,
        users: [],
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.addRole(newRole).subscribe((res) => {
        expect(res).toEqual(responseRole);
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles`,
      );
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual(newRole);
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      req.flush(responseRole);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        'Role added Successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should handle error on POST failure', () => {
      const newRole = {
        name: 'NewRole',
        description: 'desc',
        status: 'ACTIVE',
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.addRole(newRole).subscribe({
        next: () => fail('Expected error'),
        error: (e) => {
          expect(e).toBe(EMPTY);
        },
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles`,
      );
      req.flush({}, { status: 400, statusText: 'Bad Request' });

      expect(mockSnackbarService.error).toHaveBeenCalled();
    });
  });

  describe('editRole', () => {
    it('should send PUT request and show success snackbar', () => {
      const roleId = 10;
      const editData = {
        name: 'Edited Role',
        description: 'updated',
        status: 'INACTIVE',
      };
      const responseRole: Role = {
        ...editData,
        id: roleId,
        users: [],
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.editRole(roleId, editData).subscribe((res) => {
        expect(res).toEqual(responseRole);
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles/${roleId}`,
      );
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual(editData);
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      req.flush(responseRole);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        'Role edited Successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should handle error on PUT failure', () => {
      const roleId = 10;
      const editData = {
        name: 'Edited Role',
        description: 'updated',
        status: 'INACTIVE',
      };
      const token = 'mockJwtToken';
      mockAuthService.getToken.mockReturnValue(token);

      service.editRole(roleId, editData).subscribe({
        next: () => fail('Expected error'),
        error: (e) => {
          expect(e).toBe(EMPTY);
        },
      });

      const req = httpTesting.expectOne(
        `${mockEnvironment.AUTH_BASE_URL}/api/roles/${roleId}`,
      );
      req.flush({}, { status: 404, statusText: 'Not Found' });

      expect(mockSnackbarService.error).toHaveBeenCalled();
    });
  });

  describe('assignUsersToRole', () => {
    const roleId = 101;
    const grantUrl = `${mockEnvironment.AUTH_BASE_URL}/api/roles/${roleId}/users/grant`;
    const revokeUrl = `${mockEnvironment.AUTH_BASE_URL}/api/roles/${roleId}/users/revoke`;
    const token = 'mockJwtToken';

    beforeEach(() => {
      mockAuthService.getToken.mockReturnValue(token);
    });

    it('should call only the grant endpoint when usersToAdd is non-empty', () => {
      const usersToAdd = [1, 2, 3];
      const returnedUser: User = {
        id: 999,
        emailAddress: 'user@example.com',
        password: '',
        status: 'ACTIVE',
        roles: [],
        profile: null,
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };

      service.assignUsersToRole(roleId, usersToAdd, []).subscribe((user) => {
        expect(user).toEqual(returnedUser);
      });

      const req = httpTesting.expectOne(grantUrl);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual(usersToAdd);
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      httpTesting.expectNone(revokeUrl);

      req.flush(returnedUser);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Granted ${usersToAdd.length} user(s)`,
        {
          variant: 'filled',
          autoClose: true,
        },
        true,
      );
    });

    it('should call only the revoke endpoint when usersToRemove is non-empty', () => {
      const usersToRemove = [4, 5];
      const returnedUser: User = {
        id: 1000,
        emailAddress: 'revoked@example.com',
        password: '',
        status: 'INACTIVE',
        roles: [],
        profile: null,
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };

      service.assignUsersToRole(roleId, [], usersToRemove).subscribe((user) => {
        expect(user).toEqual(returnedUser);
      });

      const req = httpTesting.expectOne(revokeUrl);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual(usersToRemove);

      httpTesting.expectNone(grantUrl);

      req.flush(returnedUser);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Revoked ${usersToRemove.length} user(s)`,
        {
          variant: 'filled',
          autoClose: true,
        },
        true,
      );
    });

    it('should call both endpoints and return the revoke result', () => {
      const usersToAdd = [11];
      const usersToRemove = [22];
      const userAfterGrant: User = {
        id: 1,
        emailAddress: 'grant@example.com',
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
        id: 2,
        emailAddress: 'revoke@example.com',
        password: '',
        status: 'INACTIVE',
        roles: [],
        profile: null,
        createdByUserId: 1,
        createdTime: '',
        modifiedByUserId: 1,
        modifiedTime: '',
      };

      service
        .assignUsersToRole(roleId, usersToAdd, usersToRemove)
        .subscribe((user) => {
          expect(user).toEqual(userAfterRevoke);
        });

      const requests = httpTesting.match(
        (req) => req.url === grantUrl || req.url === revokeUrl,
      );
      expect(requests.length).toBe(2);

      const grantReq = requests.find((r) => r.request.url === grantUrl)!;
      const revokeReq = requests.find((r) => r.request.url === revokeUrl)!;

      expect(grantReq.request.method).toBe('PUT');
      expect(grantReq.request.body).toEqual(usersToAdd);

      expect(revokeReq.request.method).toBe('PUT');
      expect(revokeReq.request.body).toEqual(usersToRemove);

      grantReq.flush(userAfterGrant);
      revokeReq.flush(userAfterRevoke);

      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Granted ${usersToAdd.length} user(s)`,
        {
          variant: 'filled',
          autoClose: true,
        },
        true,
      );
      expect(mockSnackbarService.success).toHaveBeenCalledWith(
        `Revoked ${usersToRemove.length} user(s)`,
        {
          variant: 'filled',
          autoClose: true,
        },
        true,
      );
    });

    it('should return EMPTY when no users are added or removed', () => {
      let called = false;

      service.assignUsersToRole(roleId, [], []).subscribe({
        next: () => (called = true),
        complete: () => (called = false), // Will complete immediately since EMPTY emits nothing
      });

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
