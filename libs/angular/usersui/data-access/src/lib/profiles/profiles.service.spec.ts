import { TestBed } from '@angular/core/testing';

import { ProfilesService } from './profiles.service';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { AuthService, SnackbarService } from '@jdw/angular-shared-data-access';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { EMPTY } from 'rxjs';
import { AddProfile, EditProfile, Profile } from '@jdw/angular-usersui-util';

const authServiceMock = {
  getToken: jest.fn(),
};

const snackbarServiceMock = {
  error: jest.fn(),
  success: jest.fn(),
};

const environmentMock = {
  AUTH_BASE_URL: 'http://localhost:8080',
};

describe('ProfilesService', () => {
  let service: ProfilesService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: AuthService,
          useValue: authServiceMock,
        },
        {
          provide: SnackbarService,
          useValue: snackbarServiceMock,
        },
        {
          provide: ENVIRONMENT,
          useValue: environmentMock,
        },
      ],
    });
    service = TestBed.inject(ProfilesService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('getProfiles', () => {
    it('should send a GET request with the correct headers and return profiles', () => {
      const mockProfiles: Profile[] = [
        {
          id: 1,
          firstName: 'John',
          middleName: null,
          lastName: 'Doe',
          birthdate: '1990-01-01',
          userId: 1,
          addresses: [],
          icon: null,
          createdByUserId: 1,
          createdTime: '2024-12-01T00:00:00Z',
          modifiedByUserId: 1,
          modifiedTime: '2024-12-01T00:00:00Z',
        },
        {
          id: 2,
          firstName: 'Jane',
          middleName: 'A.',
          lastName: 'Smith',
          birthdate: '1992-06-15',
          userId: 2,
          addresses: [],
          icon: null,
          createdByUserId: 2,
          createdTime: '2024-12-02T00:00:00Z',
          modifiedByUserId: 2,
          modifiedTime: '2024-12-02T00:00:00Z',
        },
      ];
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getProfiles().subscribe((profiles) => {
        expect(profiles).toEqual(mockProfiles);
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(mockProfiles);
    });

    it('should handle errors and call handleError', () => {
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getProfiles().subscribe({
        next: () => fail('Expected an error, but got success'),
        error: (error) => {
          expect(error).toBe(EMPTY);
        },
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles`,
      );
      req.flush(
        { message: 'Server Error' },
        { status: 500, statusText: 'Internal Server Error' },
      );

      expect(snackbarServiceMock.error).toHaveBeenCalledWith(
        'An unexpected error occurred on our server. Please try again later.',
        { variant: 'filled', autoClose: false },
        true,
      );
    });
  });

  describe('getProfile', () => {
    it('should fetch a single profile by userId', () => {
      const mockProfile: Profile = {
        id: 1,
        firstName: 'John',
        middleName: null,
        lastName: 'Doe',
        birthdate: '1990-01-01',
        userId: 1,
        addresses: [],
        icon: null,
        createdByUserId: 1,
        createdTime: '2024-12-01T00:00:00Z',
        modifiedByUserId: 1,
        modifiedTime: '2024-12-01T00:00:00Z',
      };
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getProfile('1').subscribe((profile) => {
        expect(profile).toEqual(mockProfile);
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/user/1`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(mockProfile);
    });
  });

  describe('addProfile', () => {
    it('should send a POST request to add a new profile', () => {
      const newProfile: AddProfile = {
        firstName: 'Alice',
        middleName: 'B.',
        lastName: 'Johnson',
        birthdate: '1995-08-21',
        userId: 3,
      };
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.addProfile(newProfile).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles`,
      );
      expect(req.request.method).toBe('POST');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      expect(req.request.body).toEqual(newProfile);

      req.flush({ ...newProfile, id: 3 });
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Profile created successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });
  });

  describe('editProfile', () => {
    it('should send a PUT request to edit a profile', () => {
      const updatedProfile: EditProfile = {
        firstName: 'Alice',
        middleName: 'C.',
        lastName: 'Johnson',
        birthdate: '1995-08-21',
      };
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.editProfile(3, updatedProfile).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/user/3`,
      );
      expect(req.request.method).toBe('PUT');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      expect(req.request.body).toEqual(updatedProfile);

      req.flush({ ...updatedProfile, id: 3 });
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Profile edited successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });
  });

  describe('handleError', () => {
    it('should display a snackbar error for 500 errors', () => {
      const mockError = new HttpErrorResponse({
        status: 500,
        error: { message: 'Internal Server Error' },
      });

      service.handleError(mockError);

      expect(snackbarServiceMock.error).toHaveBeenCalledWith(
        'An unexpected error occurred on our server. Please try again later.',
        { variant: 'filled', autoClose: false },
        true,
      );
    });

    it('should handle network errors gracefully', () => {
      const mockError = new HttpErrorResponse({
        error: new ErrorEvent('NetworkError', {
          message: 'Failed to connect to the server',
        }),
      });

      service.handleError(mockError);

      expect(snackbarServiceMock.error).toHaveBeenCalledWith(
        'An error occurred. Please contact support if the issue persists.',
        { variant: 'filled', autoClose: false },
        true,
      );
    });
  });
});
