import { ComponentFixture, TestBed } from '@angular/core/testing';
import { UsersComponent } from './users.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { UsersService } from '@jdw/angular-usersui-data-access';
import { of } from 'rxjs';
import {
  dateFilterComparator,
  dateSortComparator,
  User,
} from '@jdw/angular-usersui-util';
import { UsersActionsButtonCellRendererComponent } from '../users-actions-button-cell-renderer/users-actions-button-cell-renderer.component';

describe('UsersComponent', () => {
  let component: UsersComponent;
  let fixture: ComponentFixture<UsersComponent>;
  let usersService: UsersService;

  const mockUsers: User[] = [
    {
      id: 1,
      emailAddress: 'user1@jdwkube.com',
      password: 'P@ssw0rd',
      status: 'ACTIVE',
      roles: [],
      profile: null,
      createdByUserId: 1,
      createdTime: '2024-08-09T01:02:34.567+00:00',
      modifiedByUserId: 1,
      modifiedTime: '2024-08-09T01:02:34.567+00:00',
    },
    {
      id: 2,
      emailAddress: 'user2@jdwkube.com',
      password: 'P@ssw0rd',
      status: 'ACTIVE',
      roles: [],
      profile: null,
      createdByUserId: 1,
      createdTime: '2024-08-09T01:02:34.567+00:00',
      modifiedByUserId: 1,
      modifiedTime: '2024-08-09T01:02:34.567+00:00',
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UsersComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: UsersService,
          useValue: {
            getUsers: jest.fn(() => of(mockUsers)),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(UsersComponent);
    component = fixture.componentInstance;
    usersService = TestBed.inject(UsersService);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should call getUsers on initialization and populate users', () => {
    expect(usersService.getUsers).toHaveBeenCalled();
    expect(component.users).toEqual(mockUsers);
  });

  it('should have the correct column definitions', () => {
    const expectedColDefs = [
      { field: 'id', headerName: 'ID', cellDataType: 'number' },
      { field: 'emailAddress', headerName: 'Email', cellDataType: 'text' },
      { field: 'status', headerName: 'Status', cellDataType: 'text' },
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
          comparator: dateFilterComparator,
        },
        comparator: dateSortComparator,
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
          comparator: dateFilterComparator,
        },
        comparator: dateSortComparator,
        cellDataType: 'text',
      },
      {
        field: 'actions',
        headerName: 'Actions',
        cellRenderer: UsersActionsButtonCellRendererComponent,
        maxWidth: 72,
        minWidth: 72,
        resizable: false,
        filter: false,
        sortable: false,
      },
    ];

    expect(component.colDefs).toEqual(expectedColDefs);
  });
});
