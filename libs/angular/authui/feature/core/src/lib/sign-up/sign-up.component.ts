import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  AbstractControl,
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  ValidationErrors,
  ValidatorFn,
  Validators,
} from '@angular/forms';
import {
  CONFIRM_PASSWORD_REQUIRED_VALIDATION_MESSAGE,
  EMAIL_PATTERN_VALIDATION_MESSAGE,
  EMAIL_REQUIRED_VALIDATION_MESSAGE,
  EMAIL_VALIDATOR_PATTERN,
  PASSWORD_MATCH_VALIDATION_MESSAGE,
  PASSWORD_MIN_LENGTH_VALIDATION_MESSAGE,
  PASSWORD_PATTERN_VALIDATION_MESSAGE,
  PASSWORD_REQUIRED_VALIDATION_MESSAGE,
  PASSWORD_VALIDATOR_PATTERN,
} from '@jdw/angular-authui-util';
import { Router, RouterLink } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { MatCardModule } from '@angular/material/card';
import { MAT_FORM_FIELD_DEFAULT_OPTIONS } from '@angular/material/form-field';

@Component({
  selector: 'lib-sign-up',
  standalone: true,
  imports: [
    CommonModule,
    MatInputModule,
    MatCardModule,
    RouterLink,
    MatButtonModule,
    MatIconModule,
    ReactiveFormsModule,
  ],
  providers: [
    {
      provide: MAT_FORM_FIELD_DEFAULT_OPTIONS,
      useValue: {
        subscriptSizing: 'dynamic',
      },
    },
  ],
  templateUrl: './sign-up.component.html',
  styleUrl: './sign-up.component.scss',
})
export class SignUpComponent {
  hide = true;
  signUpForm = new FormGroup({
    email: new FormControl('', {
      nonNullable: true,
      validators: [
        Validators.required,
        Validators.pattern(EMAIL_VALIDATOR_PATTERN),
      ],
    }),
    matchingPassword: new FormGroup(
      {
        password: new FormControl('', {
          nonNullable: true,
          validators: [
            Validators.required,
            Validators.minLength(8),
            Validators.pattern(PASSWORD_VALIDATOR_PATTERN),
          ],
        }),
        confirmPassword: new FormControl('', {
          nonNullable: true,
          validators: [
            Validators.required,
            Validators.minLength(8),
            Validators.pattern(PASSWORD_VALIDATOR_PATTERN),
          ],
        }),
      },
      { validators: this.passwordMatch() },
    ),
  });
  validationMessages = {
    email: [
      { type: 'required', message: EMAIL_REQUIRED_VALIDATION_MESSAGE },
      { type: 'pattern', message: EMAIL_PATTERN_VALIDATION_MESSAGE },
    ],
    password: [
      { type: 'required', message: PASSWORD_REQUIRED_VALIDATION_MESSAGE },
      {
        type: 'minlength',
        message: PASSWORD_MIN_LENGTH_VALIDATION_MESSAGE,
      },
      { type: 'pattern', message: PASSWORD_PATTERN_VALIDATION_MESSAGE },
    ],
    confirmPassword: [
      {
        type: 'required',
        message: CONFIRM_PASSWORD_REQUIRED_VALIDATION_MESSAGE,
      },
      {
        type: 'minlength',
        message: PASSWORD_MIN_LENGTH_VALIDATION_MESSAGE,
      },
      { type: 'pattern', message: PASSWORD_PATTERN_VALIDATION_MESSAGE },
    ],
    matchingPassword: [
      { type: 'passwordMatch', message: PASSWORD_MATCH_VALIDATION_MESSAGE },
    ],
  };
  private router = inject(Router);

  signUp() {
    const emailControl = this.signUpForm.get('email');
    const matchingPasswordControl = this.signUpForm.get('matchingPassword');
    if (
      emailControl &&
      matchingPasswordControl &&
      emailControl.valid &&
      matchingPasswordControl.valid
    ) {
      const email = emailControl.value;
      const passwordControl = matchingPasswordControl.get('password');
      if (passwordControl) {
        const password = passwordControl.value;
        // const roles: Role[] = ['user'];
        // const user: CreateUserRequest = { email, displayName, password, roles };
        // this.usersService.create(user).subscribe({
        //   next: () => {
        //     this.signUpForm.reset();
        //     this.snackbarService.success(
        //       SIGN_UP_SUCCESS_MESSAGE,
        //       {
        //         variant: 'filled',
        //         autoClose: true,
        //       },
        //       true,
        //     );
        //     this.router.navigate(['sign-in']);
        //   },
        // });
      }
    }
  }

  passwordMatch(): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
      const password = control.get('password')?.value;
      const confirmPassword = control.get('confirmPassword')?.value;
      const matches = password === confirmPassword;
      return !matches ? { passwordMatch: true } : null;
    };
  }

  getErrorMessage(formControlName: 'email' | 'matchingPassword') {
    for (const validation of this.validationMessages[formControlName]) {
      if (this.signUpForm.get(formControlName)?.hasError(validation.type)) {
        return validation.message;
      }
    }
    return '';
  }

  getPasswordErrorMessage(formControlName: 'password' | 'confirmPassword') {
    for (const validation of this.validationMessages[formControlName]) {
      if (
        this.signUpForm
          .get('matchingPassword')
          ?.get(formControlName)
          ?.hasError(validation.type)
      ) {
        return validation.message;
      }
    }
    return '';
  }
}
