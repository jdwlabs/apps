import { ComponentFixture, TestBed } from '@angular/core/testing';
import { EmailSignInComponent } from './email-sign-in.component';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatSnackBarModule } from '@angular/material/snack-bar';

describe('EmailSignInComponent', () => {
  let component: EmailSignInComponent;
  let fixture: ComponentFixture<EmailSignInComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EmailSignInComponent, NoopAnimationsModule, MatSnackBarModule],
    }).compileComponents();

    fixture = TestBed.createComponent(EmailSignInComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
