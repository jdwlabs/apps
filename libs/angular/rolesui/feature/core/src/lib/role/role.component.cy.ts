import { TestBed } from '@angular/core/testing';
import { RoleComponent } from './role.component';
import { ENVIRONMENT, Role } from '@jdw/angular-shared-util';
import { RolesDialogComponent } from '../roles-dialog/roles-dialog.component';
import { MAT_DIALOG_DATA } from '@angular/material/dialog';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ActivatedRoute } from '@angular/router';
import { RolesService } from '@jdw/angular-shared-data-access';
import { of } from 'rxjs';

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

describe(RoleComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
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
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
      ],
    }).overrideComponent(RoleComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(RoleComponent);
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
      cy.mount(RoleComponent, {
        providers: [
          {
            provide: RolesService,
            useValue: {
              getRole: () => of(mockRoleAll),
            },
          },
        ],
      });

      cy.getByCy('title').should('be.visible').and('contain.text', 'Role');

      cy.getByCy('edit-role-button')
        .should('be.visible')
        .and('contain.text', 'Edit Role')
        .and('be.enabled');
      cy.getByCy('delete-role-button')
        .should('be.visible')
        .and('contain.text', 'Delete Role')
        .and('be.enabled');

      cy.getByCy('name-label').should('be.visible').and('contain.text', 'Name');
      cy.getByCy('name-input')
        .should('be.visible')
        .and('contain.value', mockRoleAll.name);
      cy.getByCy('description-label')
        .should('be.visible')
        .and('contain.text', 'Description');
      cy.getByCy('description-input')
        .should('be.visible')
        .and('contain.value', mockRoleAll.description);
      cy.getByCy('status-label')
        .should('be.visible')
        .and('contain.text', 'Status');
      cy.getByCy('status-input')
        .should('be.visible')
        .and('contain.value', mockRoleAll.status);
      cy.getByCy('users-count-label')
        .should('be.visible')
        .and('contain.text', 'Users Count');
      cy.getByCy('users-count-input')
        .should('be.visible')
        .and('contain.value', mockRoleAll.users.length);
    });

    it(`should render the role form with no user data on ${size} screen size`, () => {
      cy.mount(RoleComponent, {
        providers: [
          {
            provide: RolesService,
            useValue: {
              getRole: () => of(mockRoleNoUsers),
            },
          },
        ],
      });

      cy.getByCy('title').should('be.visible').and('contain.text', 'Role');

      cy.getByCy('edit-role-button')
        .should('be.visible')
        .and('contain.text', 'Edit Role')
        .and('be.enabled');
      cy.getByCy('delete-role-button')
        .should('be.visible')
        .and('contain.text', 'Delete Role')
        .and('be.enabled');

      cy.getByCy('name-label').should('be.visible').and('contain.text', 'Name');
      cy.getByCy('name-input')
        .should('be.visible')
        .and('contain.value', mockRoleNoUsers.name);
      cy.getByCy('description-label')
        .should('be.visible')
        .and('contain.text', 'Description');
      cy.getByCy('description-input')
        .should('be.visible')
        .and('contain.value', mockRoleNoUsers.description);
      cy.getByCy('status-label')
        .should('be.visible')
        .and('contain.text', 'Status');
      cy.getByCy('status-input')
        .should('be.visible')
        .and('contain.value', mockRoleNoUsers.status);
      cy.getByCy('users-count-label')
        .should('be.visible')
        .and('contain.text', 'Users Count');
      cy.getByCy('users-count-input')
        .should('be.visible')
        .and('contain.value', mockRoleNoUsers.users.length);
    });
  });
}
