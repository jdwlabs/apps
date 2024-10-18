import { inject, Injectable } from '@angular/core';
import {
  HttpClient,
  HttpErrorResponse,
  HttpHeaders,
} from '@angular/common/http';
import { SnackbarService } from '../snackbar/snackbar.service';
import { AuthService } from '../auth/auth.service';
import {
  ENVIRONMENT,
  Environment,
  getErrorMessage,
  Role,
} from '@jdw/angular-shared-util';
import { catchError, EMPTY, Observable } from 'rxjs';

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
