import { inject, Injectable } from '@angular/core';
import {
  HttpClient,
  HttpErrorResponse,
  HttpHeaders,
} from '@angular/common/http';
import { AuthService, SnackbarService } from '@jdw/angular-shared-data-access';
import {
  ENVIRONMENT,
  Environment,
  getErrorMessage,
} from '@jdw/angular-shared-util';
import { catchError, EMPTY, map, Observable, tap } from 'rxjs';
import {
  AddProfile,
  Address,
  AddressRequest,
  EditProfile,
  Profile,
} from '@jdw/angular-usersui-util';

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
        `${this.environment.AUTH_BASE_URL}/api/profiles/user/${userId}`,
        {
          headers: headers,
        },
      )
      .pipe(catchError((error) => this.handleError(error)));
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
        `${this.environment.AUTH_BASE_URL}/api/profiles/user/${userId}`,
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

  getAddress(profileId: number, addressId: number): Observable<Address> {
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
        map((profile) => {
          const address = profile.addresses.find(
            (addr) => addr.id === addressId,
          );
          if (!address) {
            throw new HttpErrorResponse({
              status: 404,
              statusText: 'Address Not Found',
              error: {
                message: `Address with ID ${addressId} not found for profile ${profileId}`,
              },
            });
          }
          return address;
        }),
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

  addIcon(profileId: number, icon: File): Observable<Profile> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });
    const formData = new FormData();
    formData.append('icon', icon);

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
    formData.append('icon', icon);

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

  handleError(error: HttpErrorResponse) {
    const errorMessage = getErrorMessage(error);
    if (error.status != 404) {
      this.snackbarService.error(
        errorMessage,
        {
          variant: 'filled',
          autoClose: false,
        },
        true,
      );
    }
    return EMPTY;
  }
}
