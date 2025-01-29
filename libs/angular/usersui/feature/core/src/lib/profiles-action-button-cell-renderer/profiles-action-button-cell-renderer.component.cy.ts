import { TestBed } from '@angular/core/testing';
import { ProfilesActionButtonCellRendererComponent } from './profiles-action-button-cell-renderer.component';
import { ActivatedRoute } from '@angular/router';

describe(ProfilesActionButtonCellRendererComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
      ],
    }).overrideComponent(ProfilesActionButtonCellRendererComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(ProfilesActionButtonCellRendererComponent);
  });

  it('should have buttons setup and displayed correctly', () => {
    cy.mount(ProfilesActionButtonCellRendererComponent);
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
