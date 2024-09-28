import { ComponentFixture, TestBed } from '@angular/core/testing';
import { EmailSignInComponent } from './email-sign-in.component';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { ActivatedRoute } from '@angular/router';

describe('EmailSignInComponent', () => {
  let component: EmailSignInComponent;
  let fixture: ComponentFixture<EmailSignInComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EmailSignInComponent, NoopAnimationsModule, MatSnackBarModule],
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

    fixture = TestBed.createComponent(EmailSignInComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
