import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ProfilesComponent } from './profiles.component';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import { Profile } from '@jdw/angular-shared-util';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { of } from 'rxjs';
import { ProfilesActionButtonCellRendererComponent } from '../profiles-action-button-cell-renderer/profiles-action-button-cell-renderer.component';

describe('ProfilesComponent', () => {
  let component: ProfilesComponent;
  let fixture: ComponentFixture<ProfilesComponent>;
  let profilesService: ProfilesService;

  const mockProfiles: Profile[] = [
    {
      id: 1,
      firstName: 'John',
      middleName: 'A.',
      lastName: 'Doe',
      birthdate: '1990-01-01',
      userId: 101,
      addresses: [
        {
          id: 1,
          addressLine1: '123 Main St',
          addressLine2: 'Apt 4B',
          city: 'Anytown',
          stateProvince: 'CA',
          postalCode: '90210',
          country: 'USA',
          profileId: 1,
          createdByUserId: 1,
          createdTime: '2024-08-09T01:02:34.567+00:00',
          modifiedByUserId: 2,
          modifiedTime: '2024-08-10T01:02:34.567+00:00',
        },
        {
          id: 2,
          addressLine1: '456 Elm St',
          addressLine2: '',
          city: 'Somewhere',
          stateProvince: 'NY',
          postalCode: '10001',
          country: 'USA',
          profileId: 1,
          createdByUserId: 1,
          createdTime: '2024-08-09T01:02:34.567+00:00',
          modifiedByUserId: 2,
          modifiedTime: '2024-08-10T01:02:34.567+00:00',
        },
      ],
      icon: {
        id: 1,
        profileId: 1,
        icon: 'icon1.png',
        createdByUserId: 1,
        createdTime: '2024-08-09T01:02:34.567+00:00',
        modifiedByUserId: 2,
        modifiedTime: '2024-08-10T01:02:34.567+00:00',
      },
      createdByUserId: 1,
      createdTime: '2024-08-09T01:02:34.567+00:00',
      modifiedByUserId: 2,
      modifiedTime: '2024-08-10T01:02:34.567+00:00',
    },
    {
      id: 2,
      firstName: 'Jane',
      middleName: null,
      lastName: 'Smith',
      birthdate: '1985-05-15',
      userId: 102,
      addresses: [],
      icon: null,
      createdByUserId: 2,
      createdTime: '2024-08-09T01:02:34.567+00:00',
      modifiedByUserId: 3,
      modifiedTime: '2024-08-10T01:02:34.567+00:00',
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProfilesComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ProfilesService,
          useValue: {
            getProfiles: jest.fn(() => of(mockProfiles)),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ProfilesComponent);
    component = fixture.componentInstance;
    profilesService = TestBed.inject(ProfilesService);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should call getProfiles on initialization and populate profiles', () => {
    expect(profilesService.getProfiles).toHaveBeenCalled();
    expect(component.profiles).toEqual(mockProfiles);
  });

  it('should have the correct column definitions', () => {
    const expectedColDefs = [
      { field: 'id', headerName: 'ID', cellDataType: 'number' },
      { field: 'firstName', headerName: 'First Name', cellDataType: 'text' },
      { field: 'middleName', headerName: 'Middle Name', cellDataType: 'text' },
      { field: 'lastName', headerName: 'Last Name', cellDataType: 'text' },
      {
        field: 'birthdate',
        headerName: 'Birthdate',
        cellDataType: 'text',
        filter: 'agDateColumnFilter',
        filterParams: {
          comparator: expect.any(Function),
        },
      },
      { field: 'userId', headerName: 'User ID', cellDataType: 'text' },
      {
        field: 'addresses',
        headerName: 'Address Count',
        valueGetter: expect.any(Function),
        cellDataType: 'number',
      },
      {
        field: 'icon',
        headerName: 'Has Icon',
        valueGetter: expect.any(Function),
        cellDataType: 'text',
      },
      {
        field: 'createdByUserId',
        headerName: 'Created By',
        cellDataType: 'number',
      },
      {
        field: 'createdTime',
        headerName: 'Created Time',
        filter: 'agDateColumnFilter',
        filterParams: {
          comparator: expect.any(Function),
        },
        comparator: expect.any(Function),
        cellDataType: 'text',
      },
      {
        field: 'modifiedByUserId',
        headerName: 'Modified By',
        cellDataType: 'number',
      },
      {
        field: 'modifiedTime',
        headerName: 'Modified Time',
        filter: 'agDateColumnFilter',
        filterParams: {
          comparator: expect.any(Function),
        },
        comparator: expect.any(Function),
        cellDataType: 'text',
      },
      {
        field: 'actions',
        headerName: 'Actions',
        cellRenderer: ProfilesActionButtonCellRendererComponent,
        maxWidth: 172,
        minWidth: 172,
        resizable: false,
        filter: false,
        sortable: false,
      },
    ];

    expect(component.colDefs).toEqual(expectedColDefs);
  });
});
