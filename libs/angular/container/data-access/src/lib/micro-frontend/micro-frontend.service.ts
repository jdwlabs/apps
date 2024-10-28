import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import {
  Environment,
  ENVIRONMENT,
  getErrorMessage,
} from '@jdw/angular-shared-util';
import { catchError, Observable, of } from 'rxjs';
import { MicroFrontendRoute } from '@jdw/angular-container-util';
import { SnackbarService } from '@jdw/angular-shared-data-access';

@Injectable({
  providedIn: 'root',
})
export class MicroFrontendService {
  private http: HttpClient = inject(HttpClient);
  private environment: Environment = inject(ENVIRONMENT);
  private snackbarService: SnackbarService = inject(SnackbarService);

  getRoutes(): Observable<MicroFrontendRoute[]> {
    return this.http
      .get<
        MicroFrontendRoute[]
      >(`${this.environment.SERVICE_DISCOVERY_BASE_URL}/api/micro-frontends`)
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
    return of([]);
  }
}
