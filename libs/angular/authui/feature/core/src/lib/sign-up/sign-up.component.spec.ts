import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SignUpComponent } from './sign-up.component';
import { ActivatedRoute } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';

describe('SignUpComponent', () => {
  let component: SignUpComponent;
  let fixture: ComponentFixture<SignUpComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SignUpComponent, NoopAnimationsModule],
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
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(SignUpComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('form validation', () => {
    it('requires email', () => {
      component.signUpForm.get('email')!.setValue('');
      expect(component.signUpForm.get('email')!.hasError('required')).toBe(
        true,
      );
    });

    it('rejects invalid email format', () => {
      component.signUpForm.get('email')!.setValue('not-an-email');
      expect(component.signUpForm.get('email')!.hasError('pattern')).toBe(true);
    });

    it('requires password', () => {
      component.signUpForm.get('matchingPassword.password')!.setValue('');
      expect(
        component.signUpForm
          .get('matchingPassword.password')!
          .hasError('required'),
      ).toBe(true);
    });

    it('requires confirmPassword', () => {
      component.signUpForm
        .get('matchingPassword.confirmPassword')!
        .setValue('');
      expect(
        component.signUpForm
          .get('matchingPassword.confirmPassword')!
          .hasError('required'),
      ).toBe(true);
    });

    it('is invalid when empty', () => {
      expect(component.signUpForm.valid).toBe(false);
    });
  });
});
