import { TestBed } from '@angular/core/testing';
import { EmailSignInComponent } from './email-sign-in.component';
import { provideRouter, Route } from '@angular/router';
import { Component } from '@angular/core';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';

describe(EmailSignInComponent.name, () => {
  beforeEach(() => {
    @Component({
      selector: 'lib-test-home',
      standalone: true,
      template: '<div>Test Home Component Works!</div>',
    })
    class TestHomeComponent {}
    const routes: Route[] = [
      {
        path: 'home',
        component: TestHomeComponent,
      },
    ];
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule, MatSnackBarModule],
      providers: [
        provideRouter(routes),
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
      ],
    }).overrideComponent(EmailSignInComponent, {
      add: {
        imports: [MatSnackBarModule],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(EmailSignInComponent);
  });

  it('should be setup properly', () => {
    cy.mount(EmailSignInComponent);
    cy.getByCy('email-address-field')
      .should('be.visible')
      .and('contain.text', 'Enter your email');
    cy.getByCy('password-field')
      .should('be.visible')
      .and('contain.text', 'Enter your password');
    cy.getByCy('password-visibility-button')
      .should('be.visible')
      .and('be.enabled')
      .click();
    cy.getByCy('sign-in-button')
      .should('be.visible')
      .and('contain.text', 'Sign In')
      .and('be.enabled');
  });

  it('should show error fields for improper input', () => {
    cy.mount(EmailSignInComponent);
    cy.getByCy('email-address-field').find('input').click().blur();
    cy.getByCy('email-address-field')
      .should('contain.text', 'Email is required')
      .type('test')
      .should('contain.text', 'Enter a valid email');
    cy.getByCy('password-field').find('input').click().blur();
    cy.getByCy('password-field').should('contain.text', 'Password is required');
  });

  it('should submit form successfully with correct input', () => {
    cy.mount(EmailSignInComponent);
    cy.getByCy('email-address-field').type('test-user-1@usersrole.com');
    cy.getByCy('password-field').type('testPassword');
    cy.getByCy('sign-in-button').click();
    // cy.getByCy('snackbar-container')
    //   .should('be.visible')
    //   .and('contain.text', 'Sign in successful');
  });
});
