import { TestBed } from '@angular/core/testing';
import { AddressComponent } from './address.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ActivatedRoute } from '@angular/router';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';
import { Address } from '@jdw/angular-usersui-util';

const testAddress: Partial<Address> = {
  id: 1,
  addressLine1: '123 Main St',
  addressLine2: '',
  city: 'Metropolis',
  stateProvince: 'NY',
  postalCode: '12345',
  country: 'USA',
  profileId: 1,
  createdByUserId: 1,
  createdTime: '2024-08-09T01:02:34.567+00:00',
  modifiedByUserId: 1,
  modifiedTime: '2024-09-01T10:11:12.000+00:00',
};

describe(AddressComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              params: {
                profileId: 1,
                addressId: 1,
              },
            },
          },
        },
        {
          provide: ENVIRONMENT,
          useValue: ENVIRONMENT,
        },
      ],
    }).overrideComponent(AddressComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(AddressComponent);
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

    it(`should render address correctly on ${size} screen size`, () => {
      cy.mount(AddressComponent);
      cy.getByCy('title').should('be.visible').and('contain.text', 'Address');

      cy.getByCy('address-line-1-label')
        .should('be.visible')
        .and('contain.text', 'Address Line 1');
      cy.getByCy('address-line-1-field').should('be.visible');

      cy.getByCy('address-line-2-label')
        .should('be.visible')
        .and('contain.text', 'Address Line 2');
      cy.getByCy('address-line-2-field').should('be.visible');

      cy.getByCy('city-label').should('be.visible').and('contain.text', 'City');
      cy.getByCy('city-field').should('be.visible');

      cy.getByCy('state-province-label')
        .should('be.visible')
        .and('contain.text', 'State / Province');
      cy.getByCy('state-province-field').should('be.visible');

      cy.getByCy('postal-code-label')
        .should('be.visible')
        .and('contain.text', 'Postal Code');
      cy.getByCy('postal-code-field').should('be.visible');

      cy.getByCy('country-label')
        .should('be.visible')
        .and('contain.text', 'Country');
      cy.getByCy('country-field').should('be.visible');

      cy.getByCy('submit-button')
        .should('be.visible')
        .and('contain.text', 'Submit')
        .and('not.be.enabled');

      cy.getByCy('reset-button')
        .should('be.visible')
        .and('contain.text', 'Reset')
        .and('be.enabled');
    });

    it(`should render profile correctly in edit mode on ${size} screen size`, () => {
      TestBed.configureTestingModule({
        providers: [
          {
            provide: ProfilesService,
            useValue: {
              getAddress: (_: number, __: number) => of(testAddress),
            },
          },
        ],
      });
      cy.mount(AddressComponent);

      cy.getByCy('title')
        .should('be.visible')
        .and('contain.text', 'Edit Address');

      cy.getByCy('address-line-1-field')
        .should('be.visible')
        .and('contain.text', 'Address Line 1')
        .find('input')
        .should('have.value', testAddress.addressLine1);

      cy.getByCy('address-line-2-field')
        .should('be.visible')
        .and('contain.text', 'Address Line 2')
        .find('input')
        .should('have.value', testAddress.addressLine2);

      cy.getByCy('city-field')
        .should('be.visible')
        .and('contain.text', 'City')
        .find('input')
        .should('have.value', testAddress.city);

      cy.getByCy('state-province-field')
        .should('be.visible')
        .and('contain.text', 'State / Province')
        .find('input')
        .should('have.value', testAddress.stateProvince);

      cy.getByCy('postal-code-field')
        .should('be.visible')
        .and('contain.text', 'Postal Code')
        .find('input')
        .should('have.value', testAddress.postalCode);

      cy.getByCy('country-field')
        .should('be.visible')
        .and('contain.text', 'Country')
        .find('input')
        .should('have.value', testAddress.country);

      cy.getByCy('submit-button')
        .should('be.visible')
        .and('contain.text', 'Submit')
        .and('be.enabled');

      cy.getByCy('reset-button')
        .should('be.visible')
        .and('contain.text', 'Reset')
        .and('be.enabled');
    });

    it(`should show error messages on ${size} screen size`, () => {
      cy.mount(AddressComponent);

      cy.getByCy('address-line-1-field').find('input').focus().blur();
      cy.getByCy('address-line-1-field').contains('Address line 1 is required');

      cy.getByCy('address-line-2-field').find('input').focus().blur();

      cy.getByCy('city-field').find('input').focus().blur();
      cy.getByCy('city-field').contains('City is required');

      cy.getByCy('state-province-field').find('input').focus().blur();
      cy.getByCy('state-province-field').contains(
        'State / province is required',
      );

      cy.getByCy('postal-code-field').find('input').focus().blur();
      cy.getByCy('postal-code-field').contains('Postal code is required');

      cy.getByCy('country-field').find('input').focus().blur();
      cy.getByCy('country-field').contains('Country is required');
    });

    it(`should reset the form on ${size} screen size`, () => {
      TestBed.configureTestingModule({
        providers: [
          {
            provide: ProfilesService,
            useValue: {
              getAddress: (_: number, __: number) => of(testAddress),
            },
          },
        ],
      });
      cy.mount(AddressComponent);

      cy.getByCy('address-line-1-field')
        .find('input')
        .clear()
        .type('123 New St')
        .focus()
        .blur();
      cy.getByCy('address-line-2-field')
        .find('input')
        .clear()
        .type('Apt 456')
        .focus()
        .blur();
      cy.getByCy('city-field')
        .find('input')
        .clear()
        .type('New York')
        .focus()
        .blur();
      cy.getByCy('state-province-field')
        .find('input')
        .clear()
        .type('NY')
        .focus()
        .blur();
      cy.getByCy('postal-code-field')
        .find('input')
        .clear()
        .type('10001')
        .focus()
        .blur();
      cy.getByCy('country-field')
        .find('input')
        .clear()
        .type('USA')
        .focus()
        .blur();

      cy.getByCy('reset-button').click();

      cy.getByCy('address-line-1-field')
        .find('input')
        .and('contain.value', testAddress.addressLine1);
      cy.getByCy('address-line-2-field')
        .find('input')
        .and('contain.value', testAddress.addressLine2);
      cy.getByCy('city-field')
        .find('input')
        .and('contain.value', testAddress.city);
      cy.getByCy('state-province-field')
        .find('input')
        .and('contain.value', testAddress.stateProvince);
      cy.getByCy('postal-code-field')
        .find('input')
        .and('contain.value', testAddress.postalCode);
      cy.getByCy('country-field')
        .find('input')
        .and('contain.value', testAddress.country);
    });
  });
}
