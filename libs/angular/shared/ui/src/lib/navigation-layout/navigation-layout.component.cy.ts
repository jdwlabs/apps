import { TestBed } from '@angular/core/testing';
import { NavigationLayoutComponent } from './navigation-layout.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ActivatedRoute } from '@angular/router';

describe(NavigationLayoutComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
    }).overrideComponent(NavigationLayoutComponent, {
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
    cy.mount(NavigationLayoutComponent);
  });

  it('should be setup properly with single navigation item properties', () => {
    cy.mount(NavigationLayoutComponent, {
      componentProperties: {
        navigationItems: [
          {
            path: '',
            icon: 'home',
            title: 'Home',
          },
        ],
      },
    });
    cy.getByCy('sidenav-container').should('be.visible');
    cy.getByCy('sidenav').should('be.visible');
    cy.getByCy('sidenav-content').should('be.visible');
    cy.getByCy('nav-list').should('be.visible');
    cy.getByCy('home-navigation-item').should('be.visible');
    cy.getByCy('navigation-title').should('not.exist');
    cy.getByCy('expand-toggle-button').should('be.visible').click();
    cy.getByCy('navigation-title')
      .should('be.visible')
      .and('contain.text', 'Home');
  });

  it('should be setup properly with multiple navigation items properties', () => {
    cy.mount(NavigationLayoutComponent, {
      componentProperties: {
        navigationItems: [
          {
            path: '',
            icon: 'bug_report',
            title: 'Debug',
          },
          {
            path: 'home',
            icon: 'home',
            title: 'Home',
          },
          {
            path: 'build',
            icon: 'build',
            title: 'Build',
          },
        ],
      },
    });
    cy.getByCy('sidenav-container').should('be.visible');
    cy.getByCy('sidenav').should('be.visible');
    cy.getByCy('sidenav-content').should('be.visible');
    cy.getByCy('nav-list').should('be.visible');
    cy.getByCy('debug-navigation-item').should('be.visible');
    cy.getByCy('home-navigation-item').should('be.visible');
    cy.getByCy('build-navigation-item').should('be.visible');
    cy.getByCy('navigation-title').should('not.exist');
    cy.getByCy('expand-toggle-button').should('be.visible').click();
    cy.getByCy('navigation-title')
      .should('be.visible')
      .and('contain.text', 'Home');
  });

  it('should be setup properly with overflow navigation items properties', () => {
    cy.fixture('navigation-items').then((navigationItems) => {
      cy.mount(NavigationLayoutComponent, {
        componentProperties: {
          navigationItems,
        },
      });
      cy.getByCy('sidenav-container').should('be.visible');
      cy.getByCy('sidenav').should('be.visible');
      cy.getByCy('sidenav-content').should('be.visible');
      cy.getByCy('nav-list').should('be.visible');
      cy.getByCy('debug-navigation-item').should('be.visible');
      cy.getByCy('home-navigation-item').should('be.visible');
      cy.getByCy('build-navigation-item').should('be.visible');
      cy.getByCy('cached-navigation-item').should('be.visible');
      cy.getByCy('calendar-navigation-item').should('be.visible');
      cy.getByCy('code-navigation-item').should('be.visible');
      cy.getByCy('compare-navigation-item').should('be.visible');
      cy.getByCy('dashboard-navigation-item').should('be.visible');
      cy.getByCy('delete-navigation-item')
        .scrollIntoView()
        .should('be.visible');
      cy.getByCy('navigation-title').should('not.exist');
      cy.getByCy('expand-toggle-button').should('be.visible').click();
      cy.getByCy('navigation-title')
        .should('be.visible')
        .and('contain.text', 'Home');
    });
  });

  it('should be setup properly with sidenav disabled properties', () => {
    cy.mount(NavigationLayoutComponent, {
      componentProperties: {
        isSideNavEnabled: false,
      },
    });
    cy.getByCy('sidenav-container').should('be.visible');
    cy.getByCy('sidenav').should('not.exist');
    cy.getByCy('sidenav-content').should('be.visible');
  });

  it('should be setup properly with sidenav mode over properties', () => {
    cy.mount(NavigationLayoutComponent, {
      componentProperties: {
        sideNavMode: 'over',
        navigationItems: [
          {
            path: '',
            icon: 'explore',
            title: 'Explore',
          },
        ],
      },
    });
    cy.getByCy('sidenav-container').should('be.visible');
    cy.getByCy('sidenav').should('be.visible');
    cy.getByCy('explore-navigation-item')
      .should('be.visible')
      .and('contain.text', 'Explore');
    cy.getByCy('expand-toggle-button').should('not.exist');
    cy.getByCy('sidenav-content').should('be.visible');
  });

  it('should be setup properly with sidenav mode push properties', () => {
    cy.mount(NavigationLayoutComponent, {
      componentProperties: {
        sideNavMode: 'over',
        navigationItems: [
          {
            path: '',
            icon: 'extension',
            title: 'Extension',
          },
        ],
      },
    });
    cy.getByCy('sidenav-container').should('be.visible');
    cy.getByCy('sidenav').should('be.visible');
    cy.getByCy('extension-navigation-item')
      .should('be.visible')
      .and('contain.text', 'Extension');
    cy.getByCy('expand-toggle-button').should('not.exist');
    cy.getByCy('sidenav-content').should('be.visible');
  });
});
