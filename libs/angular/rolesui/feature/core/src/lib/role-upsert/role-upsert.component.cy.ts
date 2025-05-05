import { TestBed } from '@angular/core/testing';
import { RoleUpsertComponent } from './role-upsert.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT, Role } from '@jdw/angular-shared-util';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { AccountComponent } from '../../../../../../usersui/feature/core/src/lib/account/account.component';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import { of } from 'rxjs';
import { ProfileComponent } from '../../../../../../usersui/feature/core/src/lib/profile/profile.component';
import { RolesService } from '@jdw/angular-shared-data-access';

const testRole: Partial<Role> = {
  name: 'TEST',
  description: 'This is a test role.',
};

describe(RoleUpsertComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: ENVIRONMENT,
        },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: convertToParamMap({ roleId: '1' }),
            },
          },
        },
      ],
    }).overrideComponent(RoleUpsertComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(RoleUpsertComponent);
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

    it(`should render role form correctly on ${size} screen size`, () => {
      cy.mount(RoleUpsertComponent);
      cy.getByCy('title').should('be.visible').and('contain.text', 'Add Role');

      cy.getByCy('name-field').should('be.visible');
      cy.getByCy('description-field').should('be.visible');

      cy.getByCy('submit-button')
        .should('be.visible')
        .and('contain.text', 'Submit')
        .and('be.disabled');

      cy.getByCy('reset-button')
        .should('be.visible')
        .and('contain.text', 'Reset')
        .and('be.enabled');
    });

    it(`should show error messages on ${size} screen size`, () => {
      cy.mount(RoleUpsertComponent);

      cy.getByCy('name-field').find('input').focus().blur();
      cy.getByCy('name-field').contains('Name is required');

      cy.getByCy('description-field').find('input').focus().blur();
      cy.getByCy('description-field').contains('Description is required');
    });

    it(`should render profile correctly in edit mode on ${size} screen size`, () => {
      TestBed.configureTestingModule({
        providers: [
          {
            provide: RolesService,
            useValue: {
              getRole: (_: any) => of(testRole),
            },
          },
        ],
      });
      cy.mount(RoleUpsertComponent);

      cy.getByCy('title').should('be.visible').and('contain.text', 'Edit Role');
      cy.getByCy('name-field')
        .should('be.visible')
        .and('contain.text', 'Enter the name')
        .find('input')
        .and('contain.value', testRole.name);
      cy.getByCy('description-field')
        .should('be.visible')
        .and('contain.text', 'Enter the description')
        .find('input')
        .and('contain.value', testRole.description);

      cy.getByCy('submit-button')
        .should('be.visible')
        .and('contain.text', 'Submit')
        .and('be.enabled');
      cy.getByCy('reset-button')
        .should('be.visible')
        .and('contain.text', 'Reset')
        .and('be.enabled');
    });

    it(`should reset the form on ${size} screen size`, () => {
      cy.mount(RoleUpsertComponent);

      cy.getByCy('name-field').find('input').type('TEST');
      cy.getByCy('description-field')
        .find('input')
        .type('This is a test role.');

      cy.getByCy('reset-button').click({ force: true });

      cy.getByCy('name-field').find('input').should('have.value', '');
      cy.getByCy('description-field').find('input').should('have.value', '');
    });
  });
}
