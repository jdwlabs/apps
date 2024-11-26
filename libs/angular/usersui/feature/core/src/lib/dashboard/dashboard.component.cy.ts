import { TestBed } from '@angular/core/testing';
import { DashboardComponent } from './dashboard.component';
import { ActivatedRoute } from '@angular/router';

describe(DashboardComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [{ provide: ActivatedRoute, useValue: ActivatedRoute }],
    }).overrideComponent(DashboardComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(DashboardComponent);
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
  describe(`${size}`, () => {
    before(() => {
      cy.viewport(width, height);
    });

    it(`should render dashboard correctly on ${size} screen size`, () => {
      cy.mount(DashboardComponent);
      cy.getByCy('title')
        .should('be.visible')
        .and('contain.text', 'Users Dashboard');
      cy.getByCy('tiles').should('be.visible');
      cy.getByCy('users-tile')
        .should('be.visible')
        .and('contain.text', 'Users')
        .and('contain.text', 'This is a page for viewing all the users');
      cy.getByCy('users-tile')
        .should('be.visible')
        .and('contain.text', 'Users')
        .and('contain.text', 'This is a page for viewing all the users');
      cy.getByCy('profiles-tile')
        .should('be.visible')
        .and('contain.text', 'Profiles')
        .and('contain.text', 'This is a page for viewing all the profiles');
      cy.getByCy('"current user-tile"')
        .should('be.visible')
        .and('contain.text', 'Current User')
        .and(
          'contain.text',
          'This is a page for viewing the current logged in user',
        );
    });
  });
}
