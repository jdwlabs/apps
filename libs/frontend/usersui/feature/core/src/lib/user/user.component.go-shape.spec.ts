import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { ENVIRONMENT } from '@jdw/frontend-shared-util';
import { UserComponent } from './user.component';

const API = 'http://localhost:8080';

const profile = {
  id: 7,
  firstName: 'Jane',
  middleName: null,
  lastName: 'Roe',
  birthdate: '1990-01-01',
  userId: 5,
  icon: null,
  addresses: [
    {
      id: 11,
      addressLine1: '1 Main St',
      addressLine2: null,
      city: 'Metropolis',
      stateProvince: 'NY',
      postalCode: '12345',
      country: 'USA',
    },
  ],
};

// identity-service serves profileId where the JVM service embedded the profile.
const goUser = {
  id: 5,
  emailAddress: 'jane@example.com',
  status: 'ACTIVE',
  roles: [],
  profileId: 7,
};

describe('UserComponent against a user that only carries profileId', () => {
  let httpTesting: HttpTestingController;
  const navigate = vi.fn();

  function create(user: object, profileReply?: () => void) {
    TestBed.configureTestingModule({
      imports: [UserComponent, NoopAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: { navigate } },
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { paramMap: { get: () => '5' } } },
        },
        { provide: ENVIRONMENT, useValue: { AUTH_BASE_URL: API } },
      ],
    });
    httpTesting = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(UserComponent);
    fixture.detectChanges();
    httpTesting.expectOne(`${API}/api/users/5`).flush(user);
    profileReply?.();
    fixture.detectChanges();
    return fixture;
  }

  const hasDataCy = (root: HTMLElement, name: string) =>
    !!root.querySelector(`[data-cy="${name}"]`);

  it('renders the fetched profile and addresses', () => {
    const fixture = create(goUser, () =>
      httpTesting.expectOne(`${API}/api/profiles/by-user/5`).flush(profile),
    );
    const root = fixture.nativeElement as HTMLElement;
    const component = fixture.componentInstance;

    expect(component.userForm.get('profile.firstName')?.value).toBe('Jane');
    expect(component.addresses.length).toBe(1);
    expect(hasDataCy(root, 'edit-profile-button')).toBe(true);
    expect(hasDataCy(root, 'add-profile-button')).toBe(false);
  });

  it('points address and icon actions at the fetched profile id', () => {
    const fixture = create(goUser, () =>
      httpTesting.expectOne(`${API}/api/profiles/by-user/5`).flush(profile),
    );

    fixture.componentInstance.editAddress(0);
    fixture.componentInstance.addIcon();

    expect(navigate).toHaveBeenCalledWith(['../../profile/7/address/11'], {
      relativeTo: expect.anything(),
    });
    expect(navigate).toHaveBeenCalledWith(['../../profile/7/icon'], {
      relativeTo: expect.anything(),
    });
  });

  it('offers Add Profile when the profile does not exist', () => {
    const fixture = create(goUser, () =>
      httpTesting
        .expectOne(`${API}/api/profiles/by-user/5`)
        .flush('not found', { status: 404, statusText: 'Not Found' }),
    );

    expect(
      hasDataCy(fixture.nativeElement as HTMLElement, 'add-profile-button'),
    ).toBe(true);
  });

  it('asks nothing of the profile service for an embedded profile', () => {
    const fixture = create({ ...goUser, profileId: undefined, profile });

    httpTesting.expectNone(`${API}/api/profiles/by-user/5`);
    expect(
      fixture.componentInstance.userForm.get('profile.firstName')?.value,
    ).toBe('Jane');
  });

  afterEach(() => {
    navigate.mockClear();
    httpTesting.verify();
  });
});
