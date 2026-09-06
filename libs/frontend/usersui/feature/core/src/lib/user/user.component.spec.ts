import { ComponentFixture, TestBed } from '@angular/core/testing';
import { UserComponent } from './user.component';
import { User } from '@jdw/frontend-shared-util';
import { FormBuilder } from '@angular/forms';
import { UsersService } from '@jdw/frontend-shared-data-access';
import { RolesService } from '@jdw/frontend-shared-data-access';
import { ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { By } from '@angular/platform-browser';
import { ENVIRONMENT, Role } from '@jdw/frontend-shared-util';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { MatDialog } from '@angular/material/dialog';
import { ProfilesService } from '@jdw/frontend-usersui-data-access';

const mockUser: User = {
  id: 1,
  emailAddress: 'user@jdwkube.com',
  password: 'P@ssw0rd',
  status: 'ACTIVE',
  roles: [
    {
      userId: 1,
      roleId: 1,
      createdByUserId: 1,
      createdTime: '2024-09-01T10:11:12.000+00:00',
    },
  ],
  profile: {
    id: 1,
    firstName: 'John',
    middleName: 'A',
    lastName: 'Doe',
    birthdate: '1990-01-01',
    userId: 1,
    icon: {
      id: 1,
      profileId: 1,
      icon: '<base64-encode-image>',
      createdByUserId: 1,
      createdTime: '2024-08-09T01:02:34.567+00:00',
      modifiedByUserId: 1,
      modifiedTime: '2024-09-01T10:11:12.000+00:00',
    },
    addresses: [
      {
        id: 1,
        addressLine1: '123 Main St',
        addressLine2: '',
        city: 'Metropolis',
        stateProvince: 'NY',
        postalCode: '12345',
        country: 'USA',
        profileId: 1,
        createdByUserId: 1,
        createdTime: '2024-08-09T01:02:34.567+00:00',
        modifiedByUserId: 1,
        modifiedTime: '2024-09-01T10:11:12.000+00:00',
      },
    ],
    createdByUserId: 1,
    createdTime: '2024-08-09T01:02:34.567+00:00',
    modifiedByUserId: 1,
    modifiedTime: '2024-08-09T01:02:34.567+00:00',
  },
  createdByUserId: 1,
  createdTime: '2024-08-09T01:02:34.567+00:00',
  modifiedByUserId: 1,
  modifiedTime: '2024-08-09T01:02:34.567+00:00',
};

const mockRole: Role = {
  id: 1,
  name: 'Admin',
  description: 'Administrator role with full access',
  status: 'ACTIVE',
  users: [],
  createdByUserId: 1,
  createdTime: '2024-08-09T01:02:34.567+00:00',
  modifiedByUserId: 2,
  modifiedTime: '2024-09-01T10:11:12.000+00:00',
};

const mockUsersService = {
  getUser: vi.fn(() => of(mockUser)),
};

const mockRolesService = {
  getRole: vi.fn(() => of(mockRole)),
};

const mockProfilesService = {
  deleteProfile: vi.fn(() => of(undefined)),
  deleteAddress: vi.fn(() => of(undefined)),
  deleteIcon: vi.fn(() => of(undefined)),
};

// Bypasses the real MatDialog/CDK overlay so `afterClosed()` can be driven
// deterministically, as if the user had confirmed the action.
const mockDialogRef = { afterClosed: () => of(true) };
const mockDialog = {
  open: vi.fn(() => mockDialogRef),
};

describe('UserComponent', () => {
  let component: UserComponent;
  let fixture: ComponentFixture<UserComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UserComponent, NoopAnimationsModule],
      providers: [
        FormBuilder,
        { provide: UsersService, useValue: mockUsersService },
        { provide: RolesService, useValue: mockRolesService },
        { provide: ProfilesService, useValue: mockProfilesService },
        { provide: MatDialog, useValue: mockDialog },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: () => '1',
              },
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

    fixture = TestBed.createComponent(UserComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should get userId from route', () => {
    expect(component.userId).toBe('1');
  });

  it('should call getUser and populate the form', () => {
    expect(mockUsersService.getUser).toHaveBeenCalledWith('1');
    expect(component.userForm.get('emailAddress')?.value).toBe(
      'user@jdwkube.com',
    );
    expect(component.userForm.get('profile')?.get('firstName')?.value).toBe(
      'John',
    );
  });

  it('should populate roles correctly', () => {
    expect(mockRolesService.getRole).toHaveBeenCalledWith(1);
    const rolesFormArray = component.roles;
    expect(rolesFormArray.length).toBe(1);
    expect(rolesFormArray.at(0).value).toBe('Admin');
  });

  it('should populate addresses correctly', () => {
    const addressesFormArray = component.addresses;
    expect(addressesFormArray.length).toBe(1);
    expect(addressesFormArray.at(0).get('city')?.value).toBe('Metropolis');
  });

  it('should render the user form correctly in the template', () => {
    const emailInput = fixture.debugElement.query(
      By.css('input[formControlName="emailAddress"]'),
    );
    expect(emailInput.nativeElement.value).toBe('user@jdwkube.com');
  });

  describe('delete actions on a failed request', () => {
    const httpError = new HttpErrorResponse({
      status: 500,
      statusText: 'Internal Server Error',
    });

    // A real `throwError(...)` observable reports an unhandled error
    // asynchronously (RxJS schedules it via a macrotask), which a
    // synchronous `expect(...).not.toThrow()` around the call site would
    // never observe — so a regression to `.subscribe(nextFn)` (no error
    // handler) would pass silently. Capturing the exact argument passed to
    // `subscribe` instead lets the test assert directly on what the
    // component code registered: an object with a real `error` function,
    // not a bare next callback.
    function capturingObservable() {
      const captured: {
        observer?: Partial<Record<'next' | 'error', unknown>>;
      } = {};
      return {
        subscribe: (observer: unknown) => {
          captured.observer =
            typeof observer === 'function' ? { next: observer } : observer;
        },
        captured,
      };
    }

    it('deleteProfile registers a real error handler and it does not throw', () => {
      const { subscribe, captured } = capturingObservable();
      mockProfilesService.deleteProfile.mockReturnValueOnce({ subscribe });
      const reloadSpy = vi.spyOn(
        component as unknown as { reloadPage: () => void },
        'reloadPage',
      );

      component.deleteProfile();

      expect(mockProfilesService.deleteProfile).toHaveBeenCalledWith(
        mockUser.id,
      );
      expect(typeof captured.observer?.error).toBe('function');
      expect(() =>
        (captured.observer!.error as (e: unknown) => void)(httpError),
      ).not.toThrow();
      expect(reloadSpy).not.toHaveBeenCalled();
      expect(component.user).toEqual(mockUser);
    });

    it('deleteAddress registers a real error handler and it does not throw', () => {
      const { subscribe, captured } = capturingObservable();
      mockProfilesService.deleteAddress.mockReturnValueOnce({ subscribe });
      const reloadSpy = vi.spyOn(
        component as unknown as { reloadPage: () => void },
        'reloadPage',
      );

      component.deleteAddress(0);

      expect(mockProfilesService.deleteAddress).toHaveBeenCalledWith(
        mockUser.profile!.id,
        mockUser.profile!.addresses[0].id,
      );
      expect(typeof captured.observer?.error).toBe('function');
      expect(() =>
        (captured.observer!.error as (e: unknown) => void)(httpError),
      ).not.toThrow();
      expect(reloadSpy).not.toHaveBeenCalled();
      expect(component.user).toEqual(mockUser);
    });

    it('deleteIcon registers a real error handler and it does not throw', () => {
      const { subscribe, captured } = capturingObservable();
      mockProfilesService.deleteIcon.mockReturnValueOnce({ subscribe });
      const reloadSpy = vi.spyOn(
        component as unknown as { reloadPage: () => void },
        'reloadPage',
      );

      component.deleteIcon();

      expect(mockProfilesService.deleteIcon).toHaveBeenCalledWith(
        mockUser.profile!.id,
      );
      expect(typeof captured.observer?.error).toBe('function');
      expect(() =>
        (captured.observer!.error as (e: unknown) => void)(httpError),
      ).not.toThrow();
      expect(reloadSpy).not.toHaveBeenCalled();
      expect(component.user).toEqual(mockUser);
    });
  });
});
