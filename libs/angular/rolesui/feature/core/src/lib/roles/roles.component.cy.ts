import { TestBed } from '@angular/core/testing';
import { RolesComponent } from './roles.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT, Role } from '@jdw/angular-shared-util';
import { ActivatedRoute } from '@angular/router';
import { RolesService } from '@jdw/angular-shared-data-access';
import { of } from 'rxjs';

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
];

describe(RolesComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
        {
          provide: RolesService,
          useValue: {
            getRoles: () => of(mockRoles),
          },
        },
      ],
    }).overrideComponent(RolesComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(RolesComponent);
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
    });

    it(`should be setup properly on ${size} screen size`, () => {
      cy.mount(RolesComponent);
      cy.getByCy('title').should('be.visible').and('contain.text', 'Roles');
      cy.getByCy('grid').should('be.visible');
      cy.get('.ag-header-row > [col-id="id"]')
        .scrollIntoView()
        .should('be.visible')
        .and('contain.text', 'ID');
      cy.contains(mockRoles[0].id);
      cy.contains(mockRoles[1].id);
      cy.get('.ag-header-row > [col-id="name"]')
        .scrollIntoView()
        .should('be.visible')
        .and('contain.text', 'Name');
      cy.contains(mockRoles[0].name);
      cy.contains(mockRoles[1].name);
      cy.get('.ag-header-row > [col-id="description"]')
        .scrollIntoView()
        .should('be.visible')
        .and('contain.text', 'Description');
      cy.contains(mockRoles[0].description);
      cy.contains(mockRoles[1].description);
      cy.get('.ag-header-row > [col-id="status"]')
        .scrollIntoView()
        .should('be.visible')
        .and('contain.text', 'Status');
      cy.contains(mockRoles[0].status);
      cy.contains(mockRoles[1].status);
      cy.get('.ag-header-row > [col-id="users"]')
        .scrollIntoView()
        .should('be.visible')
        .and('contain.text', 'Users Count');
      cy.contains(mockRoles[0].users.length);
      cy.contains(mockRoles[1].users.length);
      cy.get('.ag-header-row > [col-id="createdByUserId"]')
        .scrollIntoView()
        .should('be.visible')
        .and('contain.text', 'Created By');
      cy.contains(mockRoles[0].createdByUserId);
      cy.contains(mockRoles[1].createdByUserId);
      cy.get('.ag-header-row > [col-id="createdTime"]')
        .scrollIntoView()
        .should('be.visible')
        .and('contain.text', 'Created Time');
      cy.contains(mockRoles[0].createdTime);
      cy.contains(mockRoles[1].createdTime);
      if (width > 600) {
        cy.get('.ag-header-row > [col-id="modifiedByUserId"]')
          .should('exist')
          .scrollIntoView()
          .should('be.visible')
          .and('contain.text', 'Modified By');
        cy.contains(mockRoles[0].modifiedByUserId);
        cy.contains(mockRoles[1].modifiedByUserId);
        cy.get('.ag-header-row > [col-id="modifiedTime"]')
          .scrollIntoView()
          .should('be.visible')
          .and('contain.text', 'Modified Time');
        cy.contains(mockRoles[0].modifiedTime);
        cy.contains(mockRoles[1].modifiedTime);
        cy.get('.ag-header-row > [col-id="actions"]')
          .scrollIntoView()
          .should('be.visible')
          .and('contain.text', 'Actions');
      }
    });

    it(`should open the delete confirmation modal when clicking delete button on ${size} screen size`, () => {
      cy.mount(RolesComponent);

      if (width > 600) {
        cy.getByCy('delete-button').first().scrollIntoView().click();

        cy.getByCy('close-button').should('be.visible');
        cy.getByCy('action-button').click();
      }
    });
  });
}
