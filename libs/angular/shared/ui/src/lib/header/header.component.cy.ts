import { TestBed } from '@angular/core/testing';
import { HeaderComponent } from './header.component';
import { ActivatedRoute } from '@angular/router';

describe(HeaderComponent.name, () => {
  beforeEach(() => {
    TestBed.overrideComponent(HeaderComponent, {
      add: {
        imports: [],
        providers: [
          {
            provide: ActivatedRoute,
            useValue: {},
          },
        ],
      },
    });
  });

  it('renders', () => {
    cy.mount(HeaderComponent);
  });

  it('should setup properly with standard properties', () => {
    cy.mount(HeaderComponent, {
      componentProperties: {
        isXSmallScreen: false,
        color: 'primary',
        appTitle: 'Angular app',
        appRouterLink: '/home',
        appTooltip: 'Home',
      },
    });
    cy.getByCy('navbar-header').should('be.visible');
    cy.getByCy('navbar-button').should('not.exist');
    cy.getByCy('navbar-home-button').should('be.visible').trigger('mouseenter');
    cy.get('[role="tooltip"]').should('contain.text', 'Home');
    cy.getByCy('navbar-home-button').trigger('mouseleave');
    cy.getByCy('navbar-actions').should('exist');
  });

  it('should setup properly with small screen properties', () => {
    cy.mount(HeaderComponent, {
      componentProperties: {
        isXSmallScreen: true,
      },
    });
    cy.getByCy('navbar-header').should('be.visible');
    cy.getByCy('navbar-button').should('be.visible').and('be.enabled');
  });

  it('should setup properly with different color properties', () => {
    cy.mount(HeaderComponent, {
      componentProperties: {
        color: 'accent',
      },
    });
    cy.getByCy('navbar-header').should('be.visible');
  });

  it('should setup properly with XSmall screen size properties', () => {
    cy.mount(HeaderComponent, {
      componentProperties: {
        isXSmallScreen: true,
      },
    });
    cy.getByCy('navbar-header').should('be.visible');
    cy.getByCy('navbar-button').should('be.visible').and('be.enabled');
  });
});
