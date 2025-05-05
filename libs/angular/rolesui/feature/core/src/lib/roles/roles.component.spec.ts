import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RolesComponent } from './roles.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { RolesService } from '@jdw/angular-shared-data-access';
import { of } from 'rxjs';
import { Role } from '@jdw/angular-shared-util';
import {
  dateFilterComparator,
  dateSortComparator,
} from '@jdw/angular-usersui-util';
import { RolesActionButtonCellRendererComponent } from '../roles-action-button-cell-renderer/roles-action-button-cell-renderer.component';

describe('RolesComponent', () => {
  let component: RolesComponent;
  let fixture: ComponentFixture<RolesComponent>;
  let rolesService: RolesService;

  const mockRoles: Role[] = [
    {
      id: 1,
      name: 'Admin',
      description: 'Administrator role',
      status: 'ACTIVE',
      users: [],
      createdByUserId: 1,
      createdTime: '2024-08-09T01:02:34.567+00:00',
      modifiedByUserId: 2,
      modifiedTime: '2024-08-10T01:02:34.567+00:00',
    },
    {
      id: 2,
      name: 'User',
      description: 'User role',
      status: 'ACTIVE',
      users: [],
      createdByUserId: 1,
      createdTime: '2024-08-09T01:02:34.567+00:00',
      modifiedByUserId: 2,
      modifiedTime: '2024-08-10T01:02:34.567+00:00',
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RolesComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: RolesService,
          useValue: {
            getRoles: jest.fn(() => of(mockRoles)),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RolesComponent);
    component = fixture.componentInstance;
    rolesService = TestBed.inject(RolesService);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should call getRoles on initialization and populate roles', () => {
    expect(rolesService.getRoles).toHaveBeenCalled();
    expect(component.roles).toEqual(mockRoles);
    expect(component.loading).toBe(false);
  });

  it('should have the correct column definitions', () => {
    const expectedColDefs = [
      { field: 'id', headerName: 'ID', cellDataType: 'number' },
      { field: 'name', headerName: 'Name', cellDataType: 'text' },
      { field: 'description', headerName: 'Description', cellDataType: 'text' },
      { field: 'status', headerName: 'Status', cellDataType: 'text' },
      {
        field: 'users',
        headerName: 'Users Count',
        valueGetter: expect.any(Function),
        cellDataType: 'number',
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
        filterParams: { comparator: dateFilterComparator },
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
        filterParams: { comparator: dateFilterComparator },
        comparator: dateSortComparator,
        cellDataType: 'text',
      },
      {
        field: 'actions',
        headerName: 'Actions',
        cellRenderer: RolesActionButtonCellRendererComponent,
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
