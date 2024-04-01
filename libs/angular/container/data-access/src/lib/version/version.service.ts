import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class VersionService {
  constructor(private httpClient: HttpClient) {}

  getVersion(): Observable<string> {
    return this.httpClient.get('/VERSION', { responseType: 'text' });
  }
}
