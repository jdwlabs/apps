import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ProfileComponent } from './profile.component';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('ProfileComponent', () => {
  let component: ProfileComponent;
  let fixture: ComponentFixture<ProfileComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProfileComponent, NoopAnimationsModule],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: convertToParamMap({ userId: '1' }),
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

    fixture = TestBed.createComponent(ProfileComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('form validation', () => {
    it('requires firstName', () => {
      component.form.get('firstName')!.setValue('');
      expect(component.form.get('firstName')!.hasError('required')).toBe(true);
    });

    it('requires lastName', () => {
      component.form.get('lastName')!.setValue('');
      expect(component.form.get('lastName')!.hasError('required')).toBe(true);
    });

    it('requires birthdate', () => {
      component.form.get('birthdate')!.setValue('');
      expect(component.form.get('birthdate')!.hasError('required')).toBe(true);
    });

    it('middleName is optional', () => {
      component.form.get('middleName')!.setValue('');
      expect(component.form.get('middleName')!.hasError('required')).toBe(
        false,
      );
    });

    it('is invalid when required fields are empty', () => {
      expect(component.form.valid).toBe(false);
    });
  });
});
