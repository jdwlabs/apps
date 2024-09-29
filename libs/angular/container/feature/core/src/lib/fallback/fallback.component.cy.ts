import { TestBed } from '@angular/core/testing';
import { FallbackComponent } from './fallback.component';
import { ActivatedRoute } from '@angular/router';

describe(FallbackComponent.name, () => {
  beforeEach(() => {
    TestBed.overrideComponent(FallbackComponent, {
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
    cy.mount(FallbackComponent, {
      componentProperties: {
        redirectLink: '/home',
        redirectIcon: 'home',
        redirectText: 'Home',
      },
    });
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
  it(`should be setup properly with default properties on ${size} screen size`, () => {
    cy.viewport(width, height);
    cy.mount(FallbackComponent);
    cy.getByCy('title')
      .should('be.visible')
      .and('have.text', 'Feature Unavailable');
    cy.getByCy('message')
      .should('be.visible')
      .and(
        'contain.text',
        'An error occurred. Please contact support if the issue persists.',
      );
    cy.getByCy('redirect-button')
      .should('be.visible')
      .and('contain.text', 'Home')
      .and('be.enabled');
    cy.getByCy('redirect-icon').should('be.visible');
    cy.getByCy('reload-button')
      .should('be.visible')
      .and('contain.text', 'Reload')
      .and('be.enabled');
    cy.getByCy('reload-icon').should('be.visible');
  });
}
