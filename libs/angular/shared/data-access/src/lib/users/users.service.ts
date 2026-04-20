import { inject, Injectable } from '@angular/core';
import {
  HttpClient,
  HttpErrorResponse,
  HttpHeaders,
} from '@angular/common/http';
import {
  ENVIRONMENT,
  Environment,
  getErrorMessage,
} from '@jdw/angular-shared-util';
import { catchError, EMPTY, forkJoin, map, Observable, tap } from 'rxjs';
import { AddUser, EditUser, User } from '@jdw/angular-shared-util';
import { AuthService } from '../auth/auth.service';
import { SnackbarService } from '../snackbar/snackbar.service';

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

  getUser(userId: string | number): Observable<User> {
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

  assignRolesToUser(
    userId: number,
    rolesToAdd: number[],
    rolesToRemove: number[],
  ): Observable<User> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });
    const base = `${this.environment.AUTH_BASE_URL}/api/users/${userId}/roles`;

    const calls: Observable<User>[] = [];

    if (rolesToAdd?.length) {
      calls.push(
        this.http
          .put<User>(`${base}/grant`, rolesToAdd, { headers })
          .pipe(
            tap(() =>
              this.snackbarService.success(
                `Granted ${rolesToAdd.length} role(s)`,
                { variant: 'filled', autoClose: true },
                true,
              ),
            ),
          ),
      );
    }

    if (rolesToRemove?.length) {
      calls.push(
        this.http
          .put<User>(`${base}/revoke`, rolesToRemove, { headers })
          .pipe(
            tap(() =>
              this.snackbarService.success(
                `Revoked ${rolesToRemove.length} role(s)`,
                { variant: 'filled', autoClose: true },
                true,
              ),
            ),
          ),
      );
    }

    if (calls.length === 0) {
      return EMPTY;
    }

    // if only one call, just return it
    if (calls.length === 1) {
      return calls[0].pipe(catchError((e) => this.handleError(e)));
    }

    // if two calls, run both in parallel and return the **second** result
    return forkJoin(calls).pipe(
      map(([_, secondResult]) => secondResult),
      catchError((e) => this.handleError(e)),
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
