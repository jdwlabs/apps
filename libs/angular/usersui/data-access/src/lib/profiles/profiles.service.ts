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
import { catchError, EMPTY, Observable, tap } from 'rxjs';
import { AddProfile, EditProfile, Profile } from '@jdw/angular-usersui-util';

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

  // TODO: add testing
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

  // TODO: add testing
  addProfile(profile: AddProfile) {
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

  // TODO: add testing
  editProfile(userId: number, profile: EditProfile) {
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
