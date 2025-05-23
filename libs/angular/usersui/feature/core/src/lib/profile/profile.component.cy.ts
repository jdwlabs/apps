import { TestBed } from '@angular/core/testing';
import { ProfileComponent } from './profile.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import { of } from 'rxjs';
import { Profile } from '@jdw/angular-shared-util';

const testProfile: Partial<Profile> = {
  firstName: 'test-first-name',
  middleName: 'test-middle-name',
  lastName: 'test-last-name',
  birthdate: '2000-01-01',
};

describe(ProfileComponent.name, () => {
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
              paramMap: convertToParamMap({ userId: '1' }),
            },
          },
        },
        {
          provide: ENVIRONMENT,
          useValue: ENVIRONMENT,
        },
      ],
    }).overrideComponent(ProfileComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(ProfileComponent);
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

    it(`should render profile correctly on ${size} screen size`, () => {
      cy.mount(ProfileComponent);
      cy.getByCy('title')
        .should('be.visible')
        .and('contain.text', 'Add Profile');
      cy.getByCy('first-name-label')
        .should('be.visible')
        .and('contain.text', 'First Name');
      cy.getByCy('middle-name-field')
        .should('be.visible')
        .and('contain.text', 'Middle Name');
      cy.getByCy('last-name-field')
        .should('be.visible')
        .and('contain.text', 'Last Name');
      cy.getByCy('birthdate-field')
        .should('be.visible')
        .and('contain.text', 'Birthdate');
      cy.getByCy('birthdate-toggle').should('be.visible');
      cy.getByCy('birthdate-hint')
        .should('be.visible')
        .and('contain.text', 'YYYY/MM/DD');
      cy.getByCy('submit-button')
        .should('be.visible')
        .and('contain.text', 'Submit')
        .and('not.be.enabled');
      cy.getByCy('reset-button')
        .should('be.visible')
        .and('contain.text', 'Reset')
        .and('be.enabled');
    });

    it(`should render profile correctly in edit mode on ${size} screen size`, () => {
      TestBed.configureTestingModule({
        providers: [
          {
            provide: ProfilesService,
            useValue: {
              getProfile: (_: any) => of(testProfile),
            },
          },
        ],
      });
      cy.mount(ProfileComponent);
      cy.getByCy('title')
        .should('be.visible')
        .and('contain.text', 'Edit Profile');
      cy.getByCy('first-name-field')
        .should('be.visible')
        .and('contain.text', 'First Name')
        .find('input')
        .and('contain.value', testProfile.firstName);
      cy.getByCy('middle-name-field')
        .should('be.visible')
        .and('contain.text', 'Middle Name')
        .find('input')
        .and('contain.value', testProfile.middleName);
      cy.getByCy('last-name-field')
        .should('be.visible')
        .and('contain.text', 'Last Name')
        .find('input')
        .and('contain.value', testProfile.lastName);
      cy.getByCy('birthdate-field')
        .should('be.visible')
        .and('contain.text', 'Birthdate')
        .find('input')
        .and('contain.value', testProfile.birthdate);
      cy.getByCy('birthdate-toggle').should('be.visible');
      cy.getByCy('birthdate-hint')
        .should('be.visible')
        .and('contain.text', 'YYYY/MM/DD');
      cy.getByCy('submit-button')
        .should('be.visible')
        .and('contain.text', 'Submit')
        .and('be.enabled');
      cy.getByCy('reset-button')
        .should('be.visible')
        .and('contain.text', 'Reset')
        .and('be.enabled');
    });

    it(`should show error messages on ${size} screen size`, () => {
      cy.mount(ProfileComponent);
      cy.getByCy('first-name-field').find('input').focus().blur();
      cy.getByCy('first-name-field').contains('First name is required');
      cy.getByCy('middle-name-field').find('input').focus().blur();
      cy.getByCy('middle-name-field').should('contain.text', '');
      cy.getByCy('last-name-field').find('input').focus().blur();
      cy.getByCy('last-name-field').contains('Last name is required');
      cy.getByCy('birthdate-field').find('input').focus().blur();
      cy.getByCy('birthdate-field').contains('Birthdate is required');
      cy.getByCy('birthdate-field').find('input').type('a').focus().blur();
      cy.getByCy('birthdate-field').contains('Birthdate must be a valid date');
      cy.getByCy('birthdate-field')
        .find('input')
        .clear()
        .type('999999-01-01')
        .focus()
        .blur();
      cy.getByCy('birthdate-field').contains('Birthdate must be a past date');
    });

    it(`should reset the form on ${size} screen size`, () => {
      TestBed.configureTestingModule({
        providers: [
          {
            provide: ProfilesService,
            useValue: {
              getProfile: (_: any) => of(testProfile),
            },
          },
        ],
      });
      cy.mount(ProfileComponent);
      cy.getByCy('first-name-field')
        .find('input')
        .clear()
        .type('1')
        .focus()
        .blur();
      cy.getByCy('middle-name-field')
        .find('input')
        .clear()
        .type('2')
        .focus()
        .blur();
      cy.getByCy('last-name-field')
        .find('input')
        .clear()
        .type('3')
        .focus()
        .blur();
      cy.getByCy('birthdate-field')
        .find('input')
        .clear()
        .type('1900-01-01')
        .focus()
        .blur();
      cy.getByCy('reset-button').click();
      cy.getByCy('first-name-field')
        .find('input')
        .and('contain.value', testProfile.firstName);
      cy.getByCy('middle-name-field')
        .find('input')
        .and('contain.value', testProfile.middleName);
      cy.getByCy('last-name-field')
        .find('input')
        .and('contain.value', testProfile.lastName);
      cy.getByCy('birthdate-field')
        .find('input')
        .and('contain.value', testProfile.birthdate);
    });

    it(`should have birthdate toggleable on ${size} screen size`, () => {
      cy.mount(ProfileComponent);
      cy.getByCy('birthdate-toggle').click();
      cy.contains('2000').click();
      cy.contains('JAN').click();
      cy.contains('1').click();
      cy.getByCy('birthdate-field')
        .find('input')
        .should('contain.value', '2000-01-01');
    });
  });
}
