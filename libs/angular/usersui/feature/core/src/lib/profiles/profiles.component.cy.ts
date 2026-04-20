import { TestBed } from '@angular/core/testing';
import { ProfilesComponent } from './profiles.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import { of } from 'rxjs';
import { ActivatedRoute } from '@angular/router';

const mockProfiles = [
  {
    id: 1000,
    firstName: 'John',
    middleName: 'A',
    lastName: 'Doe',
    birthdate: '1990-01-01',
    userId: 1001,
    addresses: [
      {
        id: 1002,
        addressLine1: '123 Main St',
        addressLine2: 'Apt 4B',
        city: 'Anytown',
        stateProvince: 'CA',
        postalCode: '90210',
        country: 'USA',
        profileId: 1,
        createdByUserId: 1,
        createdTime: '2024-08-09T01:02:34.567+00:00',
        modifiedByUserId: 2,
        modifiedTime: '2024-08-10T01:02:34.567+00:00',
      },
      {
        id: 1003,
        addressLine1: '456 Elm St',
        addressLine2: '',
        city: 'Somewhere',
        stateProvince: 'NY',
        postalCode: '10001',
        country: 'USA',
        profileId: 1000,
        createdByUserId: 1005,
        createdTime: '2024-08-09T01:02:34.567+00:00',
        modifiedByUserId: 1006,
        modifiedTime: '2024-08-10T01:02:34.567+00:00',
      },
    ],
    icon: {
      id: 1007,
      profileId: 1000,
      icon: 'icon1.png',
      createdByUserId: 1008,
      createdTime: '2024-08-09T01:02:34.567+00:00',
      modifiedByUserId: 1009,
      modifiedTime: '2024-08-10T01:02:34.567+00:00',
    },
    createdByUserId: 1010,
    createdTime: '2024-08-09T01:02:34.567+00:00',
    modifiedByUserId: 1011,
    modifiedTime: '2024-08-10T01:02:34.567+00:00',
  },
  {
    id: 2000,
    firstName: 'Jane',
    middleName: null,
    lastName: 'Smith',
    birthdate: '1985-05-15',
    userId: 2001,
    addresses: [],
    icon: null,
    createdByUserId: 2002,
    createdTime: '2024-11-09T01:02:34.567+00:00',
    modifiedByUserId: 2003,
    modifiedTime: '2024-11-10T01:02:34.567+00:00',
  },
];

describe(ProfilesComponent.name, () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
        {
          provide: ProfilesService,
          useValue: {
            getProfiles: () => of(mockProfiles),
          },
        },
      ],
    }).overrideComponent(ProfilesComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(ProfilesComponent);
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
    cy.mount(ProfilesComponent);
    cy.getByCy('title').should('be.visible').and('contain.text', 'Profiles');
    cy.getByCy('grid').should('be.visible');
    cy.get('.ag-header-row > [col-id="id"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'ID');
    cy.contains(mockProfiles[0].id);
    cy.contains(mockProfiles[1].id);
    cy.get('.ag-header-row > [col-id="firstName"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'First Name');
    cy.contains(mockProfiles[0].firstName);
    cy.contains(mockProfiles[1].firstName);
    cy.get('.ag-header-row > [col-id="middleName"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Middle Name');
    if (mockProfiles[0].middleName) {
      cy.contains(mockProfiles[0].middleName);
    }
    if (mockProfiles[1].middleName) {
      cy.contains(mockProfiles[1].middleName);
    }
    cy.get('.ag-header-row > [col-id="lastName"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Last Name');
    cy.contains(mockProfiles[0].lastName);
    cy.contains(mockProfiles[1].lastName);
    cy.get('.ag-header-row > [col-id="birthdate"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Birthdate');
    cy.contains(mockProfiles[0].birthdate);
    cy.contains(mockProfiles[1].birthdate);
    cy.get('.ag-header-row > [col-id="addresses"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Address Count');
    cy.contains('2');
    cy.contains('0');
    cy.get('.ag-header-row > [col-id="icon"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Has Icon');
    cy.contains('Yes');
    cy.contains('No');
    cy.get('.ag-header-row > [col-id="createdByUserId"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Created By');
    cy.contains(mockProfiles[0].createdByUserId);
    cy.contains(mockProfiles[1].createdByUserId);
    cy.get('.ag-header-row > [col-id="createdTime"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Created Time');
    cy.contains(mockProfiles[0].createdTime);
    cy.contains(mockProfiles[1].createdTime);
    cy.get('.ag-header-row > [col-id="modifiedByUserId"]')
      .should('exist')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Modified By');
    cy.contains(mockProfiles[0].modifiedByUserId);
    cy.contains(mockProfiles[1].modifiedByUserId);
    cy.get('.ag-header-row > [col-id="modifiedTime"]')
      .scrollIntoView()
      .should('be.visible')
      .and('contain.text', 'Modified Time');
    cy.contains(mockProfiles[0].modifiedTime);
    cy.contains(mockProfiles[1].modifiedTime);
    cy.get('.ag-header-row > [col-id="actions"]')
      .should('be.visible')
      .and('contain.text', 'Actions');
    cy.getByCy('view-button')
      .first()
      .scrollIntoView()
      .should('be.visible')
      .and('be.enabled');
  });
}
