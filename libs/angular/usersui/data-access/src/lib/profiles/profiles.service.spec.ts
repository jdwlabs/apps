import { TestBed } from '@angular/core/testing';

import { ProfilesService } from './profiles.service';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { AuthService, SnackbarService } from '@jdw/angular-shared-data-access';
import {
  AddProfile,
  AddressRequest,
  EditProfile,
  ENVIRONMENT,
  Profile,
} from '@jdw/angular-shared-util';
import { EMPTY } from 'rxjs';

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

  describe('getAddress', () => {
    it('should fetch an address for the given profile and address ID', () => {
      const profileId = 1;
      const addressId = 1;
      const mockProfile: Profile = {
        id: 1,
        firstName: 'John',
        middleName: '',
        lastName: 'Doe',
        birthdate: '1990-01-01',
        userId: 1,
        addresses: [
          {
            id: 1,
            addressLine1: '123 Main St',
            addressLine2: '',
            city: 'Springfield',
            stateProvince: 'IL',
            postalCode: '62704',
            country: 'USA',
            profileId: 1,
            createdByUserId: 1,
            createdTime: '2024-12-01T00:00:00Z',
            modifiedByUserId: 1,
            modifiedTime: '2024-12-01T00:00:00Z',
          },
        ],
        icon: null,
        createdByUserId: 1,
        createdTime: '2024-12-01T00:00:00Z',
        modifiedByUserId: 1,
        modifiedTime: '2024-12-01T00:00:00Z',
      };

      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getAddress(profileId, addressId).subscribe((address) => {
        expect(address).toEqual(mockProfile.addresses[0]);
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(mockProfile);
    });

    it('should handle address not found (404) error', () => {
      const profileId = 1;
      const addressId = 999; // Address ID that doesn't exist
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getAddress(profileId, addressId).subscribe({
        next: () => fail('Expected an error, but got success'),
        error: (error) => {
          expect(error).toBeInstanceOf(HttpErrorResponse);
          expect(error.status).toBe(404);
          expect(error.error.message).toBe(
            `Address with ID ${addressId} not found for profile ${profileId}`,
          );
        },
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(
        { message: 'Address Not Found' },
        { status: 404, statusText: 'Not Found' },
      );
    });

    it('should handle unexpected HTTP errors', () => {
      const profileId = 1;
      const addressId = 1;
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getAddress(profileId, addressId).subscribe({
        next: () => fail('Expected an error, but got success'),
        error: (error) => {
          expect(error).toBeInstanceOf(HttpErrorResponse);
          expect(error.status).toBe(500);
        },
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(
        { message: 'Internal Server Error' },
        { status: 500, statusText: 'Internal Server Error' },
      );
    });
  });

  describe('addAddress', () => {
    it('should send a POST request to add an address', () => {
      const profileId = 1;
      const address: AddressRequest = {
        addressLine1: '123 Main St',
        addressLine2: 'Apt 4B',
        city: 'Springfield',
        stateProvince: 'IL',
        postalCode: '62704',
        country: 'USA',
      };
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.addAddress(profileId, address).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}/address`,
      );
      expect(req.request.method).toBe('POST');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      expect(req.request.body).toEqual(address);

      req.flush({ id: profileId, ...address });
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Address added successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });
  });

  describe('editAddress', () => {
    it('should send a PUT request to update an address', () => {
      const profileId = 1;
      const addressId = 1;
      const address: AddressRequest = {
        addressLine1: '456 Elm St',
        addressLine2: '',
        city: 'Shelbyville',
        stateProvince: 'IL',
        postalCode: '62565',
        country: 'USA',
      };
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.editAddress(profileId, addressId, address).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}/address/${addressId}`,
      );
      expect(req.request.method).toBe('PUT');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      expect(req.request.body).toEqual(address);

      req.flush({ id: profileId, ...address });
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Address updated successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });
  });

  describe('deleteAddress', () => {
    it('should send a DELETE request to remove an address', () => {
      const profileId = 1;
      const addressId = 1;
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.deleteAddress(profileId, addressId).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}/address/${addressId}`,
      );
      expect(req.request.method).toBe('DELETE');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(null, { status: 204, statusText: 'No Content' });
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Address deleted successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });
  });

  describe('addIcon', () => {
    it('should send a POST request to upload an icon', () => {
      const profileId = 1;
      const iconFile = new File(['icon'], 'icon.png', { type: 'image/png' });
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.addIcon(profileId, iconFile).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}/icon`,
      );
      expect(req.request.method).toBe('POST');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      expect(req.request.body).toBeInstanceOf(FormData);

      req.flush(null);
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Icon added successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });
  });

  describe('editIcon', () => {
    it('should send a PUT request to update an icon', () => {
      const profileId = 1;
      const iconFile = new File(['icon'], 'icon.png', { type: 'image/png' });
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.editIcon(profileId, iconFile).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}/icon`,
      );
      expect(req.request.method).toBe('PUT');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
      expect(req.request.body).toBeInstanceOf(FormData);

      req.flush({ id: profileId, ...iconFile });
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Icon updated successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });
  });

  describe('deleteIcon', () => {
    it('should send a DELETE request to remove an icon', () => {
      const profileId = 1;
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.deleteIcon(profileId).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}/icon`,
      );
      expect(req.request.method).toBe('DELETE');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(null, { status: 204, statusText: 'No Content' });
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Icon deleted successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });
  });

  describe('deleteProfile', () => {
    it('should send a DELETE request to remove a profile and show success message', () => {
      const userId = 1;
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.deleteProfile(userId).subscribe();

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/user/${userId}`,
      );
      expect(req.request.method).toBe('DELETE');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(null, { status: 204, statusText: 'No Content' });
      expect(snackbarServiceMock.success).toHaveBeenCalledWith(
        'Profile deleted successfully',
        { variant: 'filled', autoClose: true },
        true,
      );
    });

    it('should handle errors and call handleError on failure', () => {
      const userId = 1;
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.deleteProfile(userId).subscribe({
        next: () => fail('Expected an error, but got success'),
        error: (error) => {
          expect(error).toBeInstanceOf(HttpErrorResponse);
          expect(error.status).toBe(500);
          expect(error.error.message).toBe('Internal Server Error');
        },
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/user/${userId}`,
      );
      req.flush(
        { message: 'Internal Server Error' },
        { status: 500, statusText: 'Internal Server Error' },
      );
    });
  });

  describe('getIcon', () => {
    it('should fetch the icon for a given profile ID', () => {
      const profileId = 1;
      const mockIcon = { url: 'http://example.com/icon.png' };
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getIcon(profileId).subscribe((icon) => {
        expect(icon).toEqual(mockIcon);
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush({ id: profileId, icon: mockIcon });
    });

    it('should handle errors and return null if profile not found', () => {
      const profileId = 1;
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getIcon(profileId).subscribe((icon) => {
        expect(icon).toBeNull();
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}`,
      );
      req.flush(
        { message: 'Profile Not Found' },
        { status: 404, statusText: 'Not Found' },
      );
    });

    it('should handle unexpected HTTP errors and call handleError', () => {
      const profileId = 1;
      const token = 'mockJwtToken';
      authServiceMock.getToken.mockReturnValue(token);

      service.getIcon(profileId).subscribe({
        next: () => fail('Expected an error, but got success'),
        error: (error) => {
          expect(error).toBeInstanceOf(HttpErrorResponse);
          expect(error.status).toBe(500);
        },
      });

      const req = httpMock.expectOne(
        `${environmentMock.AUTH_BASE_URL}/api/profiles/${profileId}`,
      );
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);

      req.flush(
        { message: 'Internal Server Error' },
        { status: 500, statusText: 'Internal Server Error' },
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
