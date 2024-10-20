import { ComponentFixture, TestBed } from '@angular/core/testing';
import { UserComponent } from './user.component';
import { User } from '@jdw/angular-usersui-util';
import { FormBuilder } from '@angular/forms';
import { UsersService } from '@jdw/angular-usersui-data-access';
import { RolesService } from '@jdw/angular-shared-data-access';
import { ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { By } from '@angular/platform-browser';
import { Role } from '@jdw/angular-shared-util';

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
  getUser: jest.fn(() => of(mockUser)),
};

const mockRolesService = {
  getRole: jest.fn(() => of(mockRole)),
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
});
