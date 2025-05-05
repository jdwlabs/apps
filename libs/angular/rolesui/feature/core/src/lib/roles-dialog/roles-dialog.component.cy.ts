import { TestBed } from '@angular/core/testing';
import { DialogData, RolesDialogComponent } from './roles-dialog.component';
import { Role } from '@jdw/angular-shared-util';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { provideHttpClient } from '@angular/common/http';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';

const mockRoleAll: Role = {
  id: 1,
  name: 'TEST',
  description: 'Test role with test access.',
  status: 'ACTIVE',
  users: [
    {
      userId: 1,
      roleId: 1,
      createdByUserId: 1,
      createdTime: '2024-09-01T10:11:12.000+00:00',
    },
  ],
  createdByUserId: 1,
  createdTime: '2024-08-09T01:02:34.567+00:00',
  modifiedByUserId: 1,
  modifiedTime: '2024-08-09T01:02:34.567+00:00',
};

const mockRoleAllData: DialogData & Role = {
  title: 'Delete Role',
  type: 'readonly',
  action: 'delete',
  ...mockRoleAll,
};

const mockRoleNoUsers: Role = {
  id: 1,
  name: 'TEST',
  description: 'Test role with test access.',
  status: 'ACTIVE',
  users: [],
  createdByUserId: 1,
  createdTime: '2024-08-09T01:02:34.567+00:00',
  modifiedByUserId: 1,
  modifiedTime: '2024-08-09T01:02:34.567+00:00',
};

const mockRoleNoUsersData: DialogData & Role = {
  title: 'Delete Role',
  type: 'readonly',
  action: 'delete',
  ...mockRoleNoUsers,
};

describe(RolesDialogComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: MAT_DIALOG_DATA,
          useValue: {},
        },
        {
          provide: MatDialogRef,
          useValue: {},
        },
      ],
    }).overrideComponent(RolesDialogComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(RolesDialogComponent);
  });

  describe('Screen Sizes', () => {
    testScreenSize('XSmall', 400, 400);
    testScreenSize('Small', 800, 800);
    testScreenSize('Medium', 1200, 900);
    testScreenSize('Large', 1600, 1080);
    testScreenSize('XLarge', 2560, 1440);
  });
});

function testScreenSize(size: string, width: number, height: number) {
  describe(`${size}`, () => {
    before(() => {
      cy.viewport(width, height);
    });

    it(`should render the role form with all data on ${size} screen size`, () => {
      cy.mount(RolesDialogComponent, {
        providers: [
          {
            provide: MAT_DIALOG_DATA,
            useValue: mockRoleAllData,
          },
        ],
      });

      cy.getByCy('title')
        .should('be.visible')
        .and('contain.text', mockRoleAllData.title);
      cy.getByCy('id-label').should('be.visible').and('contain.text', 'ID');
      cy.getByCy('id-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.id);
      cy.getByCy('name-label').should('be.visible').and('contain.text', 'Name');
      cy.getByCy('name-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.name);
      cy.getByCy('description-label')
        .should('be.visible')
        .and('contain.text', 'Description');
      cy.getByCy('description-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.description);
      cy.getByCy('status-label')
        .should('be.visible')
        .and('contain.text', 'Status');
      cy.getByCy('status-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.status);
      cy.getByCy('role-created-time-label')
        .should('be.visible')
        .and('contain.text', 'Created Time');
      cy.getByCy('role-created-time-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.createdTime);
      cy.getByCy('role-created-by-user-id-label')
        .should('be.visible')
        .and('contain.text', 'Created By User ID');
      cy.getByCy('role-created-by-user-id-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.createdByUserId);
      cy.getByCy('role-modified-time-label')
        .should('be.visible')
        .and('contain.text', 'Modified Time');
      cy.getByCy('role-modified-time-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.modifiedTime);
      cy.getByCy('role-modified-by-user-id-label')
        .should('be.visible')
        .and('contain.text', 'Modified By User ID');
      cy.getByCy('role-modified-by-user-id-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.modifiedByUserId);
      cy.getByCy('users-title')
        .should('be.visible')
        .and('contain.text', 'Users');
      cy.getByCy('user-title-0')
        .should('be.visible')
        .and('contain.text', 'User 1');
      cy.getByCy('user-user-id-label-0')
        .should('be.visible')
        .and('contain.text', 'User ID');
      cy.getByCy('user-user-id-input-0')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.users[0].userId);
      cy.getByCy('user-role-id-label-0')
        .should('be.visible')
        .and('contain.text', 'Role ID');
      cy.getByCy('user-role-id-input-0')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.users[0].roleId);
      cy.getByCy('user-created-time-label-0')
        .should('be.visible')
        .and('contain.text', 'Created Time');
      cy.getByCy('user-created-time-input-0')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.users[0].createdTime);
      cy.getByCy('user-created-by-user-id-label-0')
        .should('be.visible')
        .and('contain.text', 'Created By User ID');
      cy.getByCy('user-created-by-user-id-input-0')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.users[0].createdByUserId);
      cy.getByCy('close-button')
        .should('be.visible')
        .and('be.enabled')
        .and('contain.text', 'Cancel');
      cy.getByCy('action-button')
        .should('be.visible')
        .and('be.enabled')
        .and('contain.text', titleCase(mockRoleAllData.action));
    });

    it(`should render the role form with no user data on ${size} screen size`, () => {
      cy.mount(RolesDialogComponent, {
        providers: [
          {
            provide: MAT_DIALOG_DATA,
            useValue: mockRoleNoUsersData,
          },
        ],
      });

      cy.getByCy('title')
        .should('be.visible')
        .and('contain.text', mockRoleAllData.title);
      cy.getByCy('id-label').should('be.visible').and('contain.text', 'ID');
      cy.getByCy('id-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.id);
      cy.getByCy('name-label').should('be.visible').and('contain.text', 'Name');
      cy.getByCy('name-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.name);
      cy.getByCy('description-label')
        .should('be.visible')
        .and('contain.text', 'Description');
      cy.getByCy('description-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.description);
      cy.getByCy('status-label')
        .should('be.visible')
        .and('contain.text', 'Status');
      cy.getByCy('status-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.status);
      cy.getByCy('role-created-time-label')
        .should('be.visible')
        .and('contain.text', 'Created Time');
      cy.getByCy('role-created-time-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.createdTime);
      cy.getByCy('role-created-by-user-id-label')
        .should('be.visible')
        .and('contain.text', 'Created By User ID');
      cy.getByCy('role-created-by-user-id-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.createdByUserId);
      cy.getByCy('role-modified-time-label')
        .should('be.visible')
        .and('contain.text', 'Modified Time');
      cy.getByCy('role-modified-time-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.modifiedTime);
      cy.getByCy('role-modified-by-user-id-label')
        .should('be.visible')
        .and('contain.text', 'Modified By User ID');
      cy.getByCy('role-modified-by-user-id-input')
        .should('be.visible')
        .and('contain.value', mockRoleAllData.modifiedByUserId);
      cy.getByCy('close-button')
        .should('be.visible')
        .and('be.enabled')
        .and('contain.text', 'Cancel');
      cy.getByCy('action-button')
        .should('be.visible')
        .and('be.enabled')
        .and('contain.text', titleCase(mockRoleAllData.action));
    });
  });
}

function titleCase(str: string) {
  return str.toLowerCase().replace(/\b\w/g, function (match) {
    return match.toUpperCase();
  });
}
