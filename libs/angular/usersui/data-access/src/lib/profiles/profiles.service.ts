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
import { catchError, EMPTY, Observable } from 'rxjs';
import { Profile } from '@jdw/angular-usersui-util';

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
