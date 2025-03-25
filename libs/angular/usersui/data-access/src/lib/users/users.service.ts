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
import { AddUser, EditUser, User } from '@jdw/angular-usersui-util';

@Injectable({
  providedIn: 'root',
})
export class UsersService {
  private http: HttpClient = inject(HttpClient);
  private snackbarService = inject(SnackbarService);
  private authService = inject(AuthService);
  private environment: Environment = inject(ENVIRONMENT);

  getUsers(): Observable<User[]> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<User[]>(`${this.environment.AUTH_BASE_URL}/api/users`, {
        headers: headers,
      })
      .pipe(catchError((error) => this.handleError(error)));
  }

  getUser(userId: string): Observable<User> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<User>(`${this.environment.AUTH_BASE_URL}/api/users/${userId}`, {
        headers: headers,
      })
      .pipe(catchError((error) => this.handleError(error)));
  }

  deleteUser(userId: number): Observable<object> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .delete(`${this.environment.AUTH_BASE_URL}/api/users/${userId}`, {
        headers: headers,
      })
      .pipe(
        tap(() => {
          this.snackbarService.success(
            `User ${userId} Deleted Successfully`,
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

  addUser(user: AddUser): Observable<User> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .post<User>(`${this.environment.AUTH_BASE_URL}/api/users`, user, {
        headers: headers,
      })
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'User added Successfully',
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

  editUser(userId: number, user: EditUser): Observable<User> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .put<User>(
        `${this.environment.AUTH_BASE_URL}/api/users/${userId}`,
        user,
        {
          headers: headers,
        },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'User edited Successfully',
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
