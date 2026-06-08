import { TestBed } from '@angular/core/testing';

import { VersionService } from './version.service';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';

describe('VersionService', () => {
  let service: VersionService;
  let httpTestingController: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
    });
    service = TestBed.inject(VersionService);
    httpTestingController = TestBed.inject(HttpTestingController);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should return version', () => {
    const version = '1.0.0';

    service.getVersion().subscribe((result) => {
      expect(result).toBe(version);
    });

    const req = httpTestingController.expectOne('/VERSION');
    expect(req.request.method).toEqual('GET');

    req.flush(version, { status: 200, statusText: 'OK' });
  });
});
