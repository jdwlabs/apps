import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DialogData, ProfileDialogComponent } from './profile-dialog.component';
import { Profile } from '@jdw/angular-shared-util';
import { FormBuilder } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

const mockData: DialogData & Profile = {
  title: 'Delete User',
  type: 'readonly',
  action: 'delete',
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
};

const mockDialogRef = {
  close: jest.fn(),
};

describe('ProfileDialogComponent', () => {
  let component: ProfileDialogComponent;
  let fixture: ComponentFixture<ProfileDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProfileDialogComponent, NoopAnimationsModule],
      providers: [
        FormBuilder,
        { provide: MAT_DIALOG_DATA, useValue: mockData },
        { provide: MatDialogRef, useValue: mockDialogRef },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ProfileDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize the form correctly', () => {
    expect(component.profileForm).toBeDefined();
    expect(component.profileForm.get('id')?.value).toEqual(mockData.id);
    expect(component.addresses.length).toBe(mockData.addresses.length);
  });

  it('should call dialogRef.close() when close() is called', () => {
    component.close();
    expect(mockDialogRef.close).toHaveBeenCalled();
  });

  it('should call dialogRef.close() with form data when action() is called', () => {
    component.action();
    expect(mockDialogRef.close).toHaveBeenCalledWith(mockData);
  });
});
