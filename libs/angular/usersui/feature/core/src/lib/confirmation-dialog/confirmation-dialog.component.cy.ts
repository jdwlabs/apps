import { TestBed } from '@angular/core/testing';
import {
  ConfirmationDialogComponent,
  ConfirmationDialogData,
} from './confirmation-dialog.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';

const mockData: ConfirmationDialogData = {
  title: 'Confirmation Dialog Testing',
  content: 'Do you wish to test this dialog?',
  action: 'Yes',
};

describe(ConfirmationDialogComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: {},
        },
        {
          provide: MatDialogRef,
          useValue: {},
        },
      ],
    }).overrideComponent(ConfirmationDialogComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(ConfirmationDialogComponent);
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

    it(`should render the confirmation dialog with all data on ${size} screen size`, () => {
      cy.mount(ConfirmationDialogComponent, {
        providers: [
          {
            provide: MAT_DIALOG_DATA,
            useValue: mockData,
          },
        ],
      });

      cy.getByCy('title')
        .should('be.visible')
        .and('contain.text', mockData.title);
      cy.getByCy('content')
        .should('be.visible')
        .and('contain.text', mockData.content);
      cy.getByCy('close-button')
        .should('be.visible')
        .and('be.enabled')
        .and('contain.text', 'Cancel');
      cy.getByCy('action-button')
        .should('be.visible')
        .and('be.enabled')
        .and('contain.text', titleCase(mockData.action));
    });
  });
}

function titleCase(str: string) {
  return str.toLowerCase().replace(/\b\w/g, function (match) {
    return match.toUpperCase();
  });
}
