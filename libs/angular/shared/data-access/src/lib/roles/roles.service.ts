import { inject, Injectable } from '@angular/core';
import {
  HttpClient,
  HttpErrorResponse,
  HttpHeaders,
} from '@angular/common/http';
import { SnackbarService } from '../snackbar/snackbar.service';
import { AuthService } from '../auth/auth.service';
import { catchError, EMPTY, Observable, tap } from 'rxjs';
import {
  AddRole,
  EditRole,
  Environment,
  ENVIRONMENT,
  getErrorMessage,
  Role,
} from '@jdw/angular-shared-util';

@Injectable({
  providedIn: 'root',
})
export class RolesService {
  private http: HttpClient = inject(HttpClient);
  private snackbarService = inject(SnackbarService);
  private authService = inject(AuthService);
  private environment: Environment = inject(ENVIRONMENT);

  getRoles(): Observable<Role[]> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<
        Role[]
      >(`${this.environment.AUTH_BASE_URL}/api/roles`, { headers: headers })
      .pipe(catchError((error) => this.handleError(error)));
  }

  getRole(roleId: string): Observable<Role> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .get<Role>(`${this.environment.AUTH_BASE_URL}/api/roles/${roleId}`, {
        headers: headers,
      })
      .pipe(catchError((error) => this.handleError(error)));
  }

  deleteRole(roleId: number | string): Observable<object> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .delete(`${this.environment.AUTH_BASE_URL}/api/roles/${roleId}`, {
        headers: headers,
      })
      .pipe(
        tap(() => {
          this.snackbarService.success(
            `Role ${roleId} Deleted Successfully`,
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

  addRole(role: AddRole): Observable<Role> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .post<Role>(`${this.environment.AUTH_BASE_URL}/api/roles`, role, {
        headers: headers,
      })
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Role added Successfully',
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

  editRole(roleId: number | string, role: EditRole): Observable<Role> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({ Authorization: `Bearer ${token}` });

    return this.http
      .put<Role>(
        `${this.environment.AUTH_BASE_URL}/api/roles/${roleId}`,
        role,
        {
          headers: headers,
        },
      )
      .pipe(
        tap(() => {
          this.snackbarService.success(
            'Role edited Successfully',
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
