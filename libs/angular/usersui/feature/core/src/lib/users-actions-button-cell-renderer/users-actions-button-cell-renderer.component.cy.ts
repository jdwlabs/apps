import { TestBed } from '@angular/core/testing';
import { UsersActionsButtonCellRendererComponent } from './users-actions-button-cell-renderer.component';
import { ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';

describe(UsersActionsButtonCellRendererComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
      ],
    }).overrideComponent(UsersActionsButtonCellRendererComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(UsersActionsButtonCellRendererComponent);
  });

  it('should have buttons setup and displayed correctly', () => {
    cy.mount(UsersActionsButtonCellRendererComponent);
    cy.getByCy('view-button')
      .should('be.visible')
      .and('be.enabled')
      .trigger('mouseenter');
    cy.get('.mdc-tooltip__surface')
      .should('be.visible')
      .and('contain.text', 'View User');
    cy.getByCy('view-button').trigger('mouseleave');

    cy.getByCy('edit-button')
      .should('be.visible')
      .and('be.enabled')
      .trigger('mouseenter');
    cy.get('.mdc-tooltip__surface')
      .should('be.visible')
      .and('contain.text', 'Edit User');
    cy.getByCy('edit-button').trigger('mouseleave');

    cy.getByCy('delete-button')
      .should('be.visible')
      .and('be.enabled')
      .trigger('mouseenter');
    cy.get('.mdc-tooltip__surface')
      .should('be.visible')
      .and('contain.text', 'Delete User');
    cy.getByCy('delete-button').trigger('mouseleave');
  });
});
