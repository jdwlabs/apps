import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import {
  EMAIL_PATTERN_VALIDATION_MESSAGE,
  EMAIL_REQUIRED_VALIDATION_MESSAGE,
  EMAIL_VALIDATOR_PATTERN,
  PASSWORD_REQUIRED_VALIDATION_MESSAGE,
} from '@jdw/angular-authui-util';
import { MatInputModule } from '@angular/material/input';
import { AuthService } from '@jdw/angular-shared-data-access';
import { Router } from '@angular/router';

@Component({
  selector: 'lib-email-sign-in',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatInputModule,
    MatIconModule,
    ReactiveFormsModule,
  ],
  templateUrl: './email-sign-in.component.html',
  styleUrl: './email-sign-in.component.scss',
})
export class EmailSignInComponent {
  form: FormGroup = new FormGroup({
    emailAddress: new FormControl('', [
      Validators.required,
      Validators.pattern(EMAIL_VALIDATOR_PATTERN),
    ]),
    password: new FormControl('', [Validators.required]),
  });
  hide = true;
  validationMessages = {
    emailAddress: [
      { type: 'required', message: EMAIL_REQUIRED_VALIDATION_MESSAGE },
      { type: 'pattern', message: EMAIL_PATTERN_VALIDATION_MESSAGE },
    ],
    password: [
      { type: 'required', message: PASSWORD_REQUIRED_VALIDATION_MESSAGE },
    ],
  };
  private authService = inject(AuthService);
  private router = inject(Router);

  signIn() {
    if (this.form.valid) {
      this.authService.signIn(this.form.value).subscribe({
        next: () => {
          this.form.reset();
          this.router.navigate(['/home']);
        },
      });
    }
  }

  getErrorMessage(formControlName: 'emailAddress' | 'password') {
    for (const validation of this.validationMessages[formControlName]) {
      if (this.form.get(formControlName)?.hasError(validation.type)) {
        return validation.message;
      }
    }
    return '';
  }
}
