import { TestBed } from '@angular/core/testing';
import { NavigationItemComponent } from './navigation-item.component';
import { ActivatedRoute } from '@angular/router';

describe(NavigationItemComponent.name, () => {
  beforeEach(() => {
    TestBed.overrideComponent(NavigationItemComponent, {
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
    cy.mount(NavigationItemComponent);
  });

  it('should be setup properly with navigation item properties', () => {
    cy.mount(NavigationItemComponent, {
      componentProperties: {
        navigationItem: {
          path: '/home',
          icon: 'home',
          title: 'Home',
        },
      },
    });
    cy.getByCy('navigation-icon').should('be.visible');
    cy.getByCy('navigation-title').should('not.exist');
  });

  it('should be setup properly with navigation item and expanded properties', () => {
    cy.mount(NavigationItemComponent, {
      componentProperties: {
        navigationItem: {
          path: '/home',
          icon: 'home',
          title: 'Home',
        },
        isExpanded: true,
      },
    });
    cy.getByCy('navigation-icon').should('be.visible');
    cy.getByCy('navigation-title')
      .should('be.visible')
      .and('contain.text', 'Home');
  });
});
