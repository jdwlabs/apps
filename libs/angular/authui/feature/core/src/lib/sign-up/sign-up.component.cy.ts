import { TestBed } from '@angular/core/testing';
import { SignUpComponent } from './sign-up.component';
import { ActivatedRoute } from '@angular/router';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

describe(SignUpComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
    }).overrideComponent(SignUpComponent, {
      add: {
        imports: [],
        providers: [
          {
            provide: ActivatedRoute,
            useValue: ActivatedRoute,
          },
        ],
      },
    });
  });

  it('renders', () => {
    cy.mount(SignUpComponent);
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
      cy.mount(SignUpComponent);
      cy.getByCy('title').should('be.visible').and('contain.text', 'Sign Up');
      cy.getByCy('email-address-field')
        .should('be.visible')
        .and('contain.text', 'Enter your email');
      cy.getByCy('password-field')
        .should('be.visible')
        .and('contain.text', 'Enter your password');
      cy.getByCy('password-visibility-button')
        .should('be.visible')
        .and('be.enabled');
      cy.getByCy('confirm-password-field')
        .should('be.visible')
        .and('contain.text', 'Enter your confirm password');
      cy.getByCy('confirm-password-visibility-button')
        .should('be.visible')
        .and('be.enabled');
      cy.getByCy('sign-up-button')
        .should('be.visible')
        .and('contain.text', 'Sign Up');
      cy.getByCy('sign-in')
        .should('be.visible')
        .and('contain.text', 'Already have an account?');
      cy.getByCy('sign-in-link')
        .should('be.visible')
        .and('contain.text', 'Sign in');
    });

    it('should show correct error messages', () => {
      cy.mount(SignUpComponent);
      cy.getByCy('sign-up-button').click();
      cy.getByCy('email-address-field')
        .should('contain.text', 'Email is required')
        .type('test')
        .should('contain.text', 'Enter a valid email');
      cy.getByCy('password-field')
        .should('contain.text', 'Password is required')
        .type('test')
        .should('contain.text', 'Password must be at least 8 characters long')
        .type('12345')
        .should(
          'contain.text',
          'Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character',
        );
      cy.getByCy('confirm-password-field')
        .should('contain.text', 'Confirm password is required')
        .type('tes')
        .should('contain.text', 'Password must be at least 8 characters long')
        .type('12345')
        .should(
          'contain.text',
          'Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character',
        );
      cy.getByCy('matching-password-error')
        .should('be.visible')
        .and('contain.text', 'Passwords must match');
    });

    it('should submit form correctly', () => {
      cy.mount(SignUpComponent);
      cy.getByCy('email-address-field').type('test-user-1@usersrole.com');
      cy.getByCy('password-field').type('testPassword');
      cy.getByCy('confirm-password-field').type('testPassword');
      cy.getByCy('sign-up-button').click();
      // cy.getByCy('snackbar-container')
      //   .should('be.visible')
      //   .and('contain.text', 'Sign up successful');
    });
  });
}
