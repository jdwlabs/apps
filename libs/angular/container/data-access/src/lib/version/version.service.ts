import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class VersionService {
  private httpClient = inject(HttpClient);

  getVersion(): Observable<string> {
    return this.httpClient.get('/VERSION', { responseType: 'text' });
  }
}
