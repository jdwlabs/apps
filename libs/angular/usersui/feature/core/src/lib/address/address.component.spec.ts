import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AddressComponent } from './address.component';
import { ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('AddressComponent', () => {
  let component: AddressComponent;
  let fixture: ComponentFixture<AddressComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AddressComponent, NoopAnimationsModule],
      providers: [
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
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: ENVIRONMENT,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AddressComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
