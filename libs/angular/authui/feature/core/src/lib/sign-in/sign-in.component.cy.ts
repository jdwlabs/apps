import { TestBed } from '@angular/core/testing';
import { SignInComponent } from './sign-in.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';

describe(SignInComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
      ],
    }).overrideComponent(SignInComponent, {
      add: {
        imports: [MatSnackBarModule],
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
    cy.mount(SignInComponent);
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
  it(`should be setup properly on ${size} screen size`, () => {
    cy.viewport(width, height);
    cy.mount(SignInComponent);
    cy.getByCy('title').should('be.visible').and('contain.text', 'Sign In');
    cy.getByCy('email-sign-in').should('be.visible');
    cy.getByCy('new-user')
      .should('be.visible')
      .and('contain.text', 'New User?');
    cy.getByCy('sign-up-link')
      .should('be.visible')
      .and('contain.text', 'Sign up');
  });
}
