import { inject, Injectable } from '@angular/core';
import {
  HttpClient,
  HttpErrorResponse,
  HttpHeaders,
} from '@angular/common/http';
import { AuthService, SnackbarService } from '@jdw/frontend-shared-data-access';
import {
  ENVIRONMENT,
  Environment,
  getErrorMessage,
} from '@jdw/frontend-shared-util';
import { catchError, map, Observable, tap, throwError } from 'rxjs';
import {
  AddProfile,
  Address,
  AddressRequest,
  EditProfile,
  Icon,
  Profile,
} from '@jdw/frontend-shared-util';

@Injectable({
  providedIn: 'root',
})
export class ProfilesService {
  private http: HttpClient = inject(HttpClient);
  private snackbarService = inject(SnackbarService);
  private authService = inject(AuthService);
  private environment: Environment = inject(ENVIRONMENT);

  getProfiles(): Observable<Profile[]> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<Profile[]>(`${this.environment.AUTH_BASE_URL}/api/profiles`, {
        headers: headers,
      })
      .pipe(catchError((error) => this.handleError(error)));
  }

  getProfile(userId: string): Observable<Profile> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/by-user/${userId}`,
        {
          headers: headers,
        },
      )
      .pipe(
        catchError((error: HttpErrorResponse) =>
          // A genuine "no profile yet" 404 (ResourceNotFoundException's
          // text/plain body, per the profile-service contract) must reach the
          // caller distinguishably from a routing 404, a 401 or a 500, so the
          // profile page can show the Add form only for the former.
          isProfileNotFoundError(error)
            ? throwError(() => error)
            : this.handleError(error),
        ),
      );
  }

  addProfile(profile: AddProfile): Observable<Profile> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .post<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles`,
        {
          firstName: profile.firstName,
          middleName: profile.middleName,
          lastName: profile.lastName,
          birthdate: profile.birthdate,
          userId: profile.userId,
        },
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Profile created successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  editProfile(userId: number, profile: EditProfile): Observable<Profile> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .put<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/by-user/${userId}`,
        {
          firstName: profile.firstName,
          middleName: profile.middleName,
          lastName: profile.lastName,
          birthdate: profile.birthdate,
        },
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Profile edited successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  deleteProfile(userId: number): Observable<void> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .delete<void>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/by-user/${userId}`,
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Profile deleted successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  getAddress(
    profileId: number,
    addressId: number,
  ): Observable<Address | undefined> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/${profileId}`,
        {
          headers: headers,
        },
      )
      .pipe(
        map((profile) =>
          profile.addresses.find((addr) => addr.id === addressId),
        ),
        catchError((error) => this.handleError(error)),
      );
  }

  addAddress(profileId: number, address: AddressRequest): Observable<Profile> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .post<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/${profileId}/address`,
        {
          addressLine1: address.addressLine1,
          addressLine2: address.addressLine2,
          city: address.city,
          stateProvince: address.stateProvince,
          postalCode: address.postalCode,
          country: address.country,
        },
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Address added successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  editAddress(
    profileId: number,
    addressId: number,
    address: AddressRequest,
  ): Observable<Profile> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .put<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/${profileId}/address/${addressId}`,
        {
          addressLine1: address.addressLine1,
          addressLine2: address.addressLine2,
          city: address.city,
          stateProvince: address.stateProvince,
          postalCode: address.postalCode,
          country: address.country,
        },
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Address updated successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  deleteAddress(profileId: number, addressId: number): Observable<void> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .delete<void>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/${profileId}/address/${addressId}`,
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Address deleted successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  getIcon(profileId: number): Observable<Icon | null> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/${profileId}`,
        {
          headers: headers,
        },
      )
      .pipe(
        map((profile) => profile.icon),
        catchError((error) => this.handleError(error)),
      );
  }

  addIcon(profileId: number, icon: File): Observable<Profile> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });
    const formData = new FormData();
    formData.append('icon', icon, icon.name);

    return this.http
      .post<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/${profileId}/icon`,
        formData,
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Icon added successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  editIcon(profileId: number, icon: File): Observable<Profile> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });
    const formData = new FormData();
    formData.append('icon', icon, icon.name);

    return this.http
      .put<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/${profileId}/icon`,
        formData,
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Icon updated successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  deleteIcon(profileId: number): Observable<void> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .delete<void>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/${profileId}/icon`,
        { headers: headers },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Icon deleted successfully',
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleError(error)),
      );
  }

  handleError(error: HttpErrorResponse): Observable<never> {
    const errorMessage = getErrorMessage(error);
    this.snackbarService.error(
      errorMessage,
      {
        variant: 'filled',
        autoClose: false,
      },
      true,
    );
    return throwError(() => error);
  }
}

/**
 * A genuine "profile not found" response, per the profile-service contract:
 * `GET /api/profiles/by-user/{userId}` answers 404 with the
 * `ResourceNotFoundException` message as a `text/plain` body. Anything else
 * that happens to carry a 404 (a routing failure, a gateway error page) does
 * not match this shape and must not be read as "no profile".
 */
export function isProfileNotFoundError(error: unknown): boolean {
  return (
    error instanceof HttpErrorResponse &&
    error.status === 404 &&
    (error.headers.get('content-type') ?? '').includes('text/plain')
  );
}
