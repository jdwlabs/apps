import { TestBed } from '@angular/core/testing';
import { UsersComponent } from './users.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { of } from 'rxjs';
import { UsersService } from '@jdw/angular-usersui-data-access';

describe(UsersComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
        {
          provide: UsersService,
          useValue: {
            getUsers: () =>
              of([
                {
                  id: 3,
                  emailAddress: 'user1@jdwkube.com',
                  password: 'P@ssw0rd',
                  status: 'ACTIVE',
                  roles: [],
                  profile: null,
                  createdByUserId: 1,
                  createdTime: '2024-08-09T01:02:34.567+00:00',
                  modifiedByUserId: 2,
                  modifiedTime: '2024-08-09T10:02:34.567+00:00',
                },
                {
                  id: 6,
                  emailAddress: 'user2@jdwkube.com',
                  password: 'P@ssw0rd',
                  status: 'ACTIVE',
                  roles: [],
                  profile: null,
                  createdByUserId: 4,
                  createdTime: '2024-08-09T11:02:34.567+00:00',
                  modifiedByUserId: 5,
                  modifiedTime: '2024-08-09T12:02:34.567+00:00',
                },
              ]),
          },
        },
      ],
    }).overrideComponent(UsersComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(UsersComponent);
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
  it(`should be setup properly on ${size} screen size`, () => {
    cy.viewport(width, height);
    cy.mount(UsersComponent);
    cy.getByCy('title').should('be.visible').and('contain.text', 'Users');
    cy.getByCy('grid').should('be.visible');
    cy.get('.ag-header-row > [col-id="id"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'ID');
    cy.contains('3');
    cy.contains('6');
    cy.get('.ag-header-row > [col-id="emailAddress"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Email');
    cy.contains('user1@jdwkube.com');
    cy.contains('user2@jdwkube.com');
    cy.get('.ag-header-row > [col-id="status"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Status');
    cy.contains('ACTIVE');
    cy.get('.ag-header-row > [col-id="createdByUserId"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Created By');
    cy.contains('1');
    cy.contains('4');
    cy.get('.ag-header-row > [col-id="createdTime"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Created Time');
    cy.contains('2024-08-09T01:02:34.567+00:00');
    cy.contains('2024-08-09T11:02:34.567+00:00');
    cy.get('.ag-header-row > [col-id="modifiedByUserId"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Modified By');
    cy.contains('2');
    cy.contains('5');
    cy.get('.ag-header-row > [col-id="modifiedTime"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Modified Time');
    cy.contains('2024-08-09T10:02:34.567+00:00');
    cy.contains('2024-08-09T12:02:34.567+00:00');
  });
}
