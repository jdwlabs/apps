import { TestBed } from '@angular/core/testing';
import { ForbiddenComponent } from './forbidden.component';
import { ActivatedRoute } from '@angular/router';

describe(ForbiddenComponent.name, () => {
  beforeEach(() => {
    TestBed.overrideComponent(ForbiddenComponent, {
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
    cy.mount(ForbiddenComponent);
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
    cy.mount(ForbiddenComponent);
    cy.getByCy('number').should('be.visible').and('have.text', '403');
    cy.getByCy('title').should('be.visible').and('have.text', 'Forbidden');
    cy.getByCy('message')
      .should('be.visible')
      .and(
        'contain.text',
        'You do not have permission to access this resource.',
      );
    cy.getByCy('redirect-button')
      .should('be.visible')
      .and('contain.text', 'Home')
      .and('be.enabled');
    cy.getByCy('redirect-icon').should('be.visible');
  });
}
