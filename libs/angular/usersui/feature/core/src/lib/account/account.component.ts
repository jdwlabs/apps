import { Component, inject, OnInit } from '@angular/core';
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
} from '@jdw/angular-shared-util';
import { ActivatedRoute, Router } from '@angular/router';
import { UsersService } from '@jdw/angular-shared-data-access';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { AddUser, EditUser, User } from '@jdw/angular-shared-util';
import { MatCardModule } from '@angular/material/card';
import { finalize } from 'rxjs';

@Component({
  selector: 'lib-account',
  imports: [
    CommonModule,
    MatInputModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    ReactiveFormsModule,
  ],
  templateUrl: './account.component.html',
  styleUrl: './account.component.scss',
})
export class AccountComponent implements OnInit {
  hide = true;
  type = 'Add';
  form = new FormGroup({
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
  userId: string | null = null;
  user: User | null = null;
  private router = inject(Router);
  private usersService = inject(UsersService);
  private route = inject(ActivatedRoute);

  ngOnInit() {
    this.userId = this.route.snapshot.paramMap.get('userId');

    if (this.userId) {
      this.usersService
        .getUser(this.userId)
        .pipe(
          finalize(() => {
            if (!this.user) {
              this.type = 'Add';
              this.router.navigate(['../'], {
                relativeTo: this.route,
              });
            }
          }),
        )
        .subscribe({
          next: (response) => {
            if (response) {
              this.user = response;
              this.type = 'Edit';
              this.form.patchValue({
                email: response.emailAddress,
                matchingPassword: {
                  password: '',
                  confirmPassword: '',
                },
              });
            }
          },
        });
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
      if (this.form.get(formControlName)?.hasError(validation.type)) {
        return validation.message;
      }
    }
    return '';
  }

  getPasswordErrorMessage(formControlName: 'password' | 'confirmPassword') {
    for (const validation of this.validationMessages[formControlName]) {
      if (
        this.form
          .get('matchingPassword')
          ?.get(formControlName)
          ?.hasError(validation.type)
      ) {
        return validation.message;
      }
    }
    return '';
  }

  onSubmit(): void {
    if (this.form.valid) {
      const userId = Number(this.userId) || 0;
      if (this.type == 'Add') {
        const account: AddUser = {
          emailAddress: this.form.get('email')?.value || '',
          password:
            this.form.get('matchingPassword')?.get('password')?.value || '',
        };
        this.usersService.addUser(account).subscribe({
          next: (response) => {
            this.router.navigate([`../user/${response.id}`], {
              relativeTo: this.route,
            });
          },
        });
      } else {
        const account: EditUser = {
          emailAddress: this.form.get('email')?.value || '',
          password:
            this.form.get('matchingPassword')?.get('password')?.value || '',
        };
        this.usersService.editUser(userId, account).subscribe({
          next: (response) => {
            this.router.navigate([`../../user/${response.id}`], {
              relativeTo: this.route,
            });
          },
        });
      }
    }
  }

  onReset(): void {
    this.form.reset({
      email: this.user?.emailAddress,
      matchingPassword: {
        password: '',
        confirmPassword: '',
      },
    });
  }
}
