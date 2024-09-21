import { TestBed } from '@angular/core/testing';
import { SnackbarComponent } from './snackbar.component';
import {
  MAT_SNACK_BAR_DATA,
  MatSnackBarRef,
} from '@angular/material/snack-bar';

describe(SnackbarComponent.name, () => {
  beforeEach(() => {
    TestBed.overrideComponent(SnackbarComponent, {
      add: {
        imports: [],
        providers: [
          {
            provide: MatSnackBarRef,
            useValue: {},
          },
          {
            provide: MAT_SNACK_BAR_DATA,
            useValue: {},
          },
        ],
      },
    });
  });

  it('renders', () => {
    cy.mount(SnackbarComponent);
  });

  it('should be setup properly with no button text properties', () => {
    TestBed.overrideComponent(SnackbarComponent, {
      add: {
        providers: [
          {
            provide: MAT_SNACK_BAR_DATA,
            useValue: {
              icon: 'home',
              message: 'This is a component test',
              buttonText: undefined,
            },
          },
        ],
      },
    });
    cy.mount(SnackbarComponent);
    cy.getByCy('icon').should('be.visible');
    cy.getByCy('message')
      .should('be.visible')
      .and('contain.text', 'This is a component test');
    cy.getByCy('close-button').should('be.visible').and('be.enabled');
  });

  it('should be setup properly with button text properties', () => {
    TestBed.overrideComponent(SnackbarComponent, {
      add: {
        providers: [
          {
            provide: MAT_SNACK_BAR_DATA,
            useValue: {
              icon: 'home',
              message: 'This is a component test',
              buttonText: 'Test',
            },
          },
        ],
      },
    });
    cy.mount(SnackbarComponent);
    cy.getByCy('icon').should('be.visible');
    cy.getByCy('message')
      .should('be.visible')
      .and('contain.text', 'This is a component test');
    cy.getByCy('close-button')
      .should('be.visible')
      .and('be.enabled')
      .and('contain.text', 'Test');
  });
});
