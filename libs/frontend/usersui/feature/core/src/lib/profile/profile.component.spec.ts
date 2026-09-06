import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ProfileComponent } from './profile.component';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { HttpHeaders, provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/frontend-shared-util';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('ProfileComponent', () => {
  let component: ProfileComponent;
  let fixture: ComponentFixture<ProfileComponent>;
  let httpMock: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProfileComponent, NoopAnimationsModule],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: convertToParamMap({ userId: '1' }),
            },
          },
        },
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: ENVIRONMENT,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ProfileComponent);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
  });

  function flushProfileLookup(
    body: unknown,
    opts: { status: number; statusText: string; headers?: HttpHeaders },
  ) {
    const req = httpMock.expectOne((request) =>
      request.url.endsWith('/api/profiles/by-user/1'),
    );
    req.flush(body, opts);
  }

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('form validation', () => {
    it('requires firstName', () => {
      component.form.get('firstName')!.setValue('');
      expect(component.form.get('firstName')!.hasError('required')).toBe(true);
    });

    it('requires lastName', () => {
      component.form.get('lastName')!.setValue('');
      expect(component.form.get('lastName')!.hasError('required')).toBe(true);
    });

    it('requires birthdate', () => {
      component.form.get('birthdate')!.setValue('');
      expect(component.form.get('birthdate')!.hasError('required')).toBe(true);
    });

    it('middleName is optional', () => {
      component.form.get('middleName')!.setValue('');
      expect(component.form.get('middleName')!.hasError('required')).toBe(
        false,
      );
    });

    it('is invalid when required fields are empty', () => {
      expect(component.form.valid).toBe(false);
    });
  });

  describe('profile loading', () => {
    it('shows the Edit form when a profile already exists', () => {
      flushProfileLookup(
        {
          id: 1,
          firstName: 'John',
          middleName: null,
          lastName: 'Doe',
          birthdate: '1990-01-01',
          userId: 1,
        },
        { status: 200, statusText: 'OK' },
      );

      expect(component.type).toBe('Edit');
      expect(component.loadError).toBe(false);
      expect(component.form.get('firstName')?.value).toBe('John');
    });

    it('shows the Add form when the user has no profile yet', () => {
      // The genuine "no profile" response: a 404 with the
      // ResourceNotFoundException text/plain body.
      flushProfileLookup('Profile not found with id 1', {
        status: 404,
        statusText: 'Not Found',
        headers: new HttpHeaders({ 'content-type': 'text/plain' }),
      });

      expect(component.type).toBe('Add');
      expect(component.loadError).toBe(false);
    });

    it('shows an error state for a routing 404 with a different body', () => {
      // A gateway/routing 404 does not carry the backend's text/plain
      // ResourceNotFoundException body, so it must not be read as "no
      // profile".
      flushProfileLookup(
        { error: 'Not Found' },
        {
          status: 404,
          statusText: 'Not Found',
          headers: new HttpHeaders({ 'content-type': 'application/json' }),
        },
      );

      expect(component.type).toBe('Add');
      expect(component.loadError).toBe(true);
    });

    it('shows an error state for a 500', () => {
      flushProfileLookup(
        { message: 'Internal Server Error' },
        { status: 500, statusText: 'Internal Server Error' },
      );

      expect(component.loadError).toBe(true);
    });
  });
});
