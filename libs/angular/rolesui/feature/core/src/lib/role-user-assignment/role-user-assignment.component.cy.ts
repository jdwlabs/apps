import { TestBed } from '@angular/core/testing';
import { RoleUserAssignmentComponent } from './role-user-assignment.component';
import { ENVIRONMENT, Role, User } from '@jdw/angular-shared-util';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { RolesService, UsersService } from '@jdw/angular-shared-data-access';
import { of } from 'rxjs';

const mockUsers: User[] = [
  {
    id: 1,
    emailAddress: 'user1@jdwkube.com',
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
  {
    id: 3,
    emailAddress: 'user3@jdwkube.com',
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

const mockRoles: Role[] = [
  {
    id: 10,
    name: 'ROLE_1',
    description: 'Test role 1 with test access.',
    status: 'ACTIVE',
    users: [],
    createdByUserId: 11,
    createdTime: '2024-08-09T01:02:34.567+00:00',
    modifiedByUserId: 12,
    modifiedTime: '2024-08-09T01:02:34.567+00:00',
  },
  {
    id: 13,
    name: 'ROLE_2',
    description: 'Test role 2 with test access.',
    status: 'ACTIVE',
    users: [],
    createdByUserId: 14,
    createdTime: '2024-08-09T01:02:34.567+00:00',
    modifiedByUserId: 15,
    modifiedTime: '2024-08-09T01:02:34.567+00:00',
  },
  {
    id: 16,
    name: 'ROLE_3',
    description: 'Test role 3 with test access.',
    status: 'ACTIVE',
    users: [],
    createdByUserId: 17,
    createdTime: '2024-08-09T01:02:34.567+00:00',
    modifiedByUserId: 18,
    modifiedTime: '2024-08-09T01:02:34.567+00:00',
  },
];

describe(RoleUserAssignmentComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [RoleUserAssignmentComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
        {
          provide: RolesService,
          useValue: {
            getRoles: () => of(mockRoles),
            getRole: (id: number) =>
              of(mockRoles.find((role) => role.id === id)),
            assignUsersToRole: (
              id: number,
              usersToAdd: number[],
              rolesToRemove: number[],
            ) => of(mockRoles.find((role) => role.id === id)),
          },
        },
        {
          provide: UsersService,
          useValue: {
            getUsers: () => of(mockUsers),
          },
        },
      ],
    }).overrideComponent(RoleUserAssignmentComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(RoleUserAssignmentComponent);
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
    beforeEach(() => {
      cy.viewport(width, height);
      cy.mount(RoleUserAssignmentComponent);
    });

    it(`should render correctly on ${size} screen size`, () => {
      cy.getByCy('title').should('contain.text', 'Assign Users');

      cy.getByCy('select-role-field').should('be.visible');

      // Confirm users are rendered
      mockUsers.forEach((user) => {
        cy.contains(user.emailAddress).should('be.visible');
        cy.contains(user.status).should('be.visible');
      });
    });

    it(`should assign and remove users for a role on ${size} screen size`, () => {
      // Simulate selecting a role (first one)
      cy.getByCy('select-role-field').first().click();
      cy.contains(mockRoles[0].name).click();

      // Assign a user
      cy.get('.ag-row-first > .ag-column-first').find('input').first().check();
      cy.getByCy('submit-button').click();

      // Assert Results
      cy.getByCy('select-role-field').should('contain.text', '');
      cy.getByCy('select-role-error')
        .should('be.visible')
        .and('contain.text', 'Role is required');
      cy.get('.ag-row-first > .ag-column-first')
        .find('input')
        .should('not.be.checked');
      cy.getByCy('submit-button').should('be.disabled');
    });

    it(`should reset users for a role on ${size} screen size`, () => {
      // Simulate selecting a role (first one)
      cy.getByCy('select-role-field').first().click();
      cy.contains(mockRoles[1].name).click();

      // Assign a user
      cy.get('.ag-row-first > .ag-column-first').find('input').first().check();
      cy.getByCy('reset-button').click();

      // Assert Results
      cy.getByCy('select-role-field').should('contain.text', '');
      cy.get('.ag-row-first > .ag-column-first')
        .find('input')
        .should('not.be.checked');
      cy.getByCy('submit-button').should('be.disabled');
    });

    it(`should allow typing and selecting a role from autocomplete on ${size} screen size`, () => {
      const roleToSelect = mockRoles[0];

      // Type part of the name into the input field
      cy.getByCy('select-role-field').find('input').type('ROLE_1');

      // Wait for options to render and select the desired role
      cy.contains(roleToSelect.name).click();

      // Verify selection shows up in the field
      cy.getByCy('select-role-field')
        .find('input')
        .should('contain.value', roleToSelect.name);
    });
  });
}
