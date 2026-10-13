import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MainComponent } from './main.component';
import { ActivatedRoute } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/frontend-shared-util';
import { provideHttpClient } from '@angular/common/http';
import { AuthService } from '@jdw/frontend-shared-data-access';
import { AccountView } from '@jdw/frontend-shared-ui';
import { BehaviorSubject } from 'rxjs';

const API = 'http://localhost:8080';

const profile = {
  id: 7,
  firstName: 'Jake',
  lastName: 'Willmsen',
  userId: 42,
  icon: { icon: 'QUJD' },
};

// A user as identity-service serves it: profileId instead of an embedded profile.
const goUser = { id: 42, emailAddress: 'jake@example.com', profileId: 7 };

describe('MainComponent', () => {
  let component: MainComponent;
  let fixture: ComponentFixture<MainComponent>;
  let httpTesting: HttpTestingController;
  const token$ = new BehaviorSubject<string | null>('jwt');

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MainComponent, NoopAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: AuthService,
          useValue: {
            token$,
            isTokenExpired: () => false,
            getUserIdFromToken: () => 42,
            getToken: () => 'jwt',
            signOut: vi.fn(),
          },
        },
        {
          provide: ActivatedRoute,
          useValue: {},
        },
        {
          provide: ENVIRONMENT,
          useValue: {
            ENVIRONMENT: 'test',
            AUTH_BASE_URL: 'http://localhost:8080',
            SERVICE_DISCOVERY_BASE_URL: 'http://localhost:9000',
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(MainComponent);
    component = fixture.componentInstance;
    httpTesting = TestBed.inject(HttpTestingController);
  });

  function resolveAccountView(
    user: object,
    profileResponse?: { body: object | string; status?: number },
  ): AccountView {
    let view!: AccountView;
    component.accountView$.subscribe((v) => (view = v));
    httpTesting.expectOne(`${API}/api/users/42`).flush(user);
    if (profileResponse) {
      const request = httpTesting.expectOne(`${API}/api/profiles/by-user/42`);
      request.flush(profileResponse.body, {
        status: profileResponse.status ?? 200,
        statusText: profileResponse.status === 404 ? 'Not Found' : 'OK',
      });
    }
    return view;
  }

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('account view', () => {
    it('shows the name and icon for a user that only carries profileId', () => {
      const view = resolveAccountView(goUser, { body: profile });

      expect(view.displayName).toBe('Jake Willmsen');
      expect(view.avatarState).toBe('icon');
      expect(view.iconData).toBe('data:image/png;base64,QUJD');
    });

    it('uses an embedded profile without asking the profile service', () => {
      const view = resolveAccountView({
        id: 42,
        emailAddress: 'jake@example.com',
        profile,
      });

      expect(view.displayName).toBe('Jake Willmsen');
      httpTesting.expectNone(`${API}/api/profiles/by-user/42`);
    });

    it('falls back to the email without a request when profileId is null', () => {
      const view = resolveAccountView({
        id: 42,
        emailAddress: 'jake@example.com',
        profileId: null,
      });

      expect(view.displayName).toBe('jake@example.com');
      httpTesting.expectNone(`${API}/api/profiles/by-user/42`);
    });

    it('falls back to the email and initial when the profile is 404', () => {
      const view = resolveAccountView(goUser, {
        body: 'Profile not found with user id 42',
        status: 404,
      });

      expect(view.displayName).toBe('jake@example.com');
      expect(view.avatarState).toBe('initials');
      expect(view.initials).toBe('J');
    });
  });
});
