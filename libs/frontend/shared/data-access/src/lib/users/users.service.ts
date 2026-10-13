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
} from '@jdw/frontend-shared-util';
import {
  catchError,
  distinctUntilChanged,
  EMPTY,
  forkJoin,
  map,
  Observable,
  of,
  startWith,
  tap,
} from 'rxjs';
import { AddUser, EditUser, Profile, User } from '@jdw/frontend-shared-util';
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

  // Resolves without a request whenever the user body already answers the
  // question: the JVM service embeds the profile (or null), and the Go service
  // sends a null profileId for a user with no profile. Any failure degrades to
  // "no profile" and stays silent because the caller is decorative chrome that
  // must not blank or nag over a profile lookup. A request in flight is
  // announced with a provisional null so a slow or hung profile service never
  // holds back the caller's signed-in state; a 404 or error adds no emission.
  getUserProfile(user: User): Observable<Profile | null> {
    if (user.profile !== undefined) {
      return of(user.profile);
    }
    if (user.profileId === null) {
      return of(null);
    }
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<Profile>(
        `${this.environment.AUTH_BASE_URL}/api/profiles/by-user/${user.id}`,
        { headers: headers },
      )
      .pipe(
        catchError(() => of(null)),
        startWith(null),
        distinctUntilChanged(),
      );
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
