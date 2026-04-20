import { TestBed } from '@angular/core/testing';
import { UserRoleAssignmentComponent } from './user-role-assignment.component';
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
    users: [
      {
        userId: 1,
        roleId: 1,
        createdByUserId: 1,
        createdTime: '2024-09-01T10:11:12.000+00:00',
      },
    ],
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

describe(UserRoleAssignmentComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [UserRoleAssignmentComponent],
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
          },
        },
        {
          provide: UsersService,
          useValue: {
            getUsers: () => of(mockUsers),
            getUser: (id: number) =>
              of(mockUsers.find((user) => user.id === id)),
            assignRolesToUser: (
              id: number,
              rolesToAdd: number[],
              rolesToRemove: number[],
            ) => of(mockUsers.find((user) => user.id === id)),
          },
        },
      ],
    }).overrideComponent(UserRoleAssignmentComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(UserRoleAssignmentComponent);
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
      cy.mount(UserRoleAssignmentComponent);
    });

    it(`should render correctly on ${size} screen size`, () => {
      cy.getByCy('title').should('contain.text', 'Assign Roles');

      cy.getByCy('select-user-field').should('be.visible');

      // Confirm roles are rendered
      mockRoles.forEach((role) => {
        cy.contains(role.name).should('be.visible');
        cy.contains(role.description).should('be.visible');
      });
    });

    it(`should assign and remove roles for a user on ${size} screen size`, () => {
      // Simulate selecting a user (first one)
      cy.getByCy('select-user-field').first().click();
      cy.contains(mockUsers[0].emailAddress).click();

      // Assign a role
      cy.get('.ag-row-first > .ag-column-first').find('input').first().check();
      cy.getByCy('submit-button').click();

      // Assert Results
      cy.getByCy('select-user-field').should('contain.text', '');
      cy.getByCy('select-user-error')
        .should('be.visible')
        .and('contain.text', 'User is required');
      cy.get('.ag-row-first > .ag-column-first')
        .find('input')
        .should('not.be.checked');
      cy.getByCy('submit-button').should('be.disabled');
    });

    it(`should reset roles for a user on ${size} screen size`, () => {
      // Simulate selecting a user (first one)
      cy.getByCy('select-user-field').first().click();
      cy.contains(mockUsers[1].emailAddress).click();

      // Assign a role
      cy.get('.ag-row-first > .ag-column-first').find('input').first().check();
      cy.getByCy('reset-button').click();

      // Assert Results
      cy.getByCy('select-user-field').should('contain.text', '');
      cy.get('.ag-row-first > .ag-column-first')
        .find('input')
        .should('not.be.checked');
      cy.getByCy('submit-button').should('be.disabled');
    });

    it(`should allow typing and selecting a user from autocomplete on ${size} screen size`, () => {
      const userToSelect = mockUsers[0];

      // Type part of the email into the input field
      cy.getByCy('select-user-field').find('input').type('user1');

      // Wait for options to render and select the desired user
      cy.contains(userToSelect.emailAddress).click();

      // Verify selection shows up in the field
      cy.getByCy('select-user-field')
        .find('input')
        .should('contain.value', userToSelect.emailAddress);
    });
  });
}
