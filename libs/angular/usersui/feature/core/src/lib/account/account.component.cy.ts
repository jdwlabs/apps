import { TestBed } from '@angular/core/testing';
import { AccountComponent } from './account.component';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

describe(AccountComponent.name, () => {
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
              paramMap: convertToParamMap({ userId: '1' }),
            },
          },
        },
      ],
    }).overrideComponent(AccountComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(AccountComponent);
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

    it(`should render account form correctly on ${size} screen size`, () => {
      cy.mount(AccountComponent);
      cy.getByCy('title').should('be.visible').and('contain.text', 'Account');

      cy.getByCy('email-address-field').should('be.visible');
      cy.getByCy('password-field').should('be.visible');
      cy.getByCy('confirm-password-field').should('be.visible');

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
      cy.mount(AccountComponent);

      cy.getByCy('email-address-field').find('input').focus().blur();
      cy.getByCy('email-address-field').contains('Email is required');

      cy.getByCy('password-field').find('input').focus().blur();
      cy.getByCy('password-field').contains('Password is required');

      cy.getByCy('confirm-password-field').find('input').focus().blur();
      cy.getByCy('confirm-password-field').contains(
        'Confirm password is required',
      );
    });

    it(`should toggle password visibility on ${size} screen size`, () => {
      cy.mount(AccountComponent);

      cy.getByCy('password-visibility-button').click();
      cy.getByCy('password-field')
        .find('input')
        .should('have.attr', 'type', 'text');

      cy.getByCy('password-visibility-button').click();
      cy.getByCy('password-field')
        .find('input')
        .should('have.attr', 'type', 'password');
    });

    it(`should reset the form on ${size} screen size`, () => {
      cy.mount(AccountComponent);

      cy.getByCy('email-address-field').find('input').type('test@example.com');
      cy.getByCy('password-field').find('input').type('password123');
      cy.getByCy('confirm-password-field')
        .find('input')
        .type('password123', { force: true });

      cy.getByCy('reset-button').click({ force: true });

      cy.getByCy('email-address-field').find('input').should('have.value', '');
      cy.getByCy('password-field').find('input').should('have.value', '');
      cy.getByCy('confirm-password-field')
        .find('input')
        .should('have.value', '');
    });
  });
}
