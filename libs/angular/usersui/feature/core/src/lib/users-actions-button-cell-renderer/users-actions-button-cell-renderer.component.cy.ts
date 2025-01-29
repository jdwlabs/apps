import { TestBed } from '@angular/core/testing';
import { UsersActionsButtonCellRendererComponent } from './users-actions-button-cell-renderer.component';
import { ActivatedRoute } from '@angular/router';

describe(UsersActionsButtonCellRendererComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
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
  });
});
