import { TestBed } from '@angular/core/testing';
import { ActionsButtonCellRendererComponent } from './actions-button-cell-renderer.component';

describe(ActionsButtonCellRendererComponent.name, () => {
  beforeEach(() => {
    TestBed.overrideComponent(ActionsButtonCellRendererComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(ActionsButtonCellRendererComponent);
  });

  it('should have buttons setup and displayed correctly', () => {
    cy.mount(ActionsButtonCellRendererComponent);
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
