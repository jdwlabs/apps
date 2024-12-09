import { TestBed } from '@angular/core/testing';
import { DashboardComponent } from './dashboard.component';
import { ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { Observable, of } from 'rxjs';
import { MicroFrontendRoute } from '@jdw/angular-container-util';
import { MicroFrontendService } from '@jdw/angular-container-data-access';

const mockMicroFrontendService = {
  getRoutes: (): Observable<MicroFrontendRoute[]> => {
    return of([
      {
        name: 'authui',
        path: 'auth',
        remoteName: 'authui',
        moduleName: './Routes',
        url: 'http://localhost:4201',
        icon: 'login',
        title: 'Auth',
        description: 'This contains sign in and sign up functionality',
      },
      {
        name: 'usersui',
        path: 'users',
        remoteName: 'usersui',
        moduleName: './Routes',
        url: 'http://localhost:4202',
        icon: 'groups',
        title: 'Users',
        description:
          'This contains viewing users and managing profiles functionality',
      },
    ]);
  },
};

describe(DashboardComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: ActivatedRoute },
        { provide: ENVIRONMENT, useValue: {} },
        { provide: MicroFrontendService, useValue: mockMicroFrontendService },
      ],
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
      cy.getByCy('title').should('be.visible').and('contain.text', 'Apps');
      cy.getByCy('tiles').should('be.visible');
      cy.getByCy('auth-tile')
        .should('be.visible')
        .and('contain.text', 'Auth')
        .and('contain.text', 'This contains sign in and sign up functionality');
      cy.getByCy('users-tile')
        .should('be.visible')
        .and('contain.text', 'Users')
        .and(
          'contain.text',
          'This contains viewing users and managing profiles functionality',
        );
    });
  });
}
