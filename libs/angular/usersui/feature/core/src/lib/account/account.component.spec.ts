import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AccountComponent } from './account.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('AccountComponent', () => {
  let component: AccountComponent;
  let fixture: ComponentFixture<AccountComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AccountComponent, NoopAnimationsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: ENVIRONMENT,
        },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: convertToParamMap({ userId: '1' }),
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AccountComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('form validation', () => {
    it('requires email', () => {
      component.form.get('email')!.setValue('');
      expect(component.form.get('email')!.hasError('required')).toBe(true);
    });

    it('rejects invalid email format', () => {
      component.form.get('email')!.setValue('not-an-email');
      expect(component.form.get('email')!.hasError('pattern')).toBe(true);
    });

    it('requires password', () => {
      component.form.get('matchingPassword.password')!.setValue('');
      expect(
        component.form.get('matchingPassword.password')!.hasError('required'),
      ).toBe(true);
    });

    it('requires confirmPassword', () => {
      component.form.get('matchingPassword.confirmPassword')!.setValue('');
      expect(
        component.form
          .get('matchingPassword.confirmPassword')!
          .hasError('required'),
      ).toBe(true);
    });

    it('is invalid when empty', () => {
      expect(component.form.valid).toBe(false);
    });
  });
});
