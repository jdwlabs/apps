import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import {
  CreateUserRequest,
  getErrorMessage,
  INVALID_SIGN_IN_MESSAGE,
  LoginUserRequest,
  SIGN_UP_SUCCESS_MESSAGE,
  SUCCESS_SIGN_IN_MESSAGE,
} from '@jdw/angular-shared-util';
import { catchError, EMPTY, Observable, tap } from 'rxjs';
import { CookieService } from 'ngx-cookie-service';
import { ENVIRONMENT, Environment } from '@jdw/angular-shared-util';
import { SnackbarService } from '../snackbar/snackbar.service';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private http: HttpClient = inject(HttpClient);
  private snackbarService = inject(SnackbarService);
  private cookieService = inject(CookieService);
  private environment: Environment = inject(ENVIRONMENT);

  getToken() {
    return this.cookieService.get('jwtToken');
  }

  signIn(user: LoginUserRequest) {
    return this.http
      .post<{
        jwtToken: string;
      }>(`${this.environment.AUTH_BASE_URL}/auth/authenticate`, user)
      .pipe(
        tap((response) => {
          this.cookieService.set('jwtToken', response.jwtToken, {
            secure: true,
            path: '/',
          });
          this.snackbarService.success(
            SUCCESS_SIGN_IN_MESSAGE,
            {
              variant: 'filled',
              autoClose: true,
            },
            true,
          );
        }),
        catchError((error) => this.handleSignInError(error)),
      );
  }

  signUp(user: CreateUserRequest) {
    return this.http
      .post(`${this.environment.AUTH_BASE_URL}/auth/user`, user)
      .pipe(
        tap(() => {
          this.snackbarService.success(
            SIGN_UP_SUCCESS_MESSAGE,
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

  handleSignInError(error: HttpErrorResponse) {
    let errorMessage = getErrorMessage(error);
    if (error.status === 400 || error.status === 401) {
      errorMessage = INVALID_SIGN_IN_MESSAGE;
    }
    this.snackbarService.error(
      errorMessage,
      {
        variant: 'filled',
        autoClose: false,
      },
      true,
    );
    return EMPTY;
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
    return EMPTY;
  }
}
