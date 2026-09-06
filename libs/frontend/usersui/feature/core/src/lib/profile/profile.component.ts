import {
  Component,
  inject,
  OnInit,
  ChangeDetectionStrategy,
} from '@angular/core';

import { MatFormField, MatLabel } from '@angular/material/form-field';
import {
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { MatInputModule } from '@angular/material/input';
import { MatDatepickerModule } from '@angular/material/datepicker';
import {
  DateAdapter,
  MAT_DATE_FORMATS,
  MAT_DATE_LOCALE,
} from '@angular/material/core';
import { MatButtonModule } from '@angular/material/button';
import { ActivatedRoute, Router } from '@angular/router';
import { AddProfile, EditProfile, Profile } from '@jdw/frontend-shared-util';
import {
  isProfileNotFoundError,
  ProfilesService,
} from '@jdw/frontend-usersui-data-access';
import { MomentDateAdapter } from '@angular/material-moment-adapter';
import { dateFormats } from '@jdw/frontend-usersui-util';

@Component({
  selector: 'lib-profile',
  imports: [
    MatFormField,
    MatLabel,
    ReactiveFormsModule,
    MatInputModule,
    MatDatepickerModule,
    MatButtonModule,
  ],
  providers: [
    {
      provide: DateAdapter,
      useClass: MomentDateAdapter,
      deps: [MAT_DATE_LOCALE],
    },
    { provide: MAT_DATE_FORMATS, useValue: dateFormats },
  ],
  templateUrl: './profile.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './profile.component.scss',
})
export class ProfileComponent implements OnInit {
  startDate = new Date('01/01/2000');
  maxDate = new Date();
  type = 'Add';
  form = new FormGroup({
    firstName: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
    middleName: new FormControl('', {
      nonNullable: true,
    }),
    lastName: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
    birthdate: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
  });
  validationMessages = {
    firstName: [{ type: 'required', message: 'First name is required' }],
    middleName: [],
    lastName: [{ type: 'required', message: 'Last name is required' }],
    birthdate: [
      {
        type: 'matDatepickerParse',
        message: 'Birthdate must be a valid date',
      },
      { type: 'required', message: 'Birthdate is required' },
      {
        type: 'matDatepickerMax',
        message: 'Birthdate must be a past date',
      },
    ],
  };
  userId: string | null = null;
  profile: Profile | null = null;
  loadError = false;
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private profileService = inject(ProfilesService);

  ngOnInit() {
    this.userId = this.route.snapshot.paramMap.get('userId');

    if (this.userId) {
      this.profileService.getProfile(this.userId).subscribe({
        next: (response) => {
          this.profile = response;
          this.type = 'Edit';
          this.loadError = false;
          this.form.patchValue({
            firstName: response.firstName,
            middleName: response.middleName || '',
            lastName: response.lastName,
            birthdate: response.birthdate,
          });
        },
        error: (error) => {
          // A genuine no-profile 404 means "Add"; anything else (a routing
          // 404, a 401, a 500) is a failed lookup, not an empty one, so it
          // must not fall through to the Add form.
          if (isProfileNotFoundError(error)) {
            this.type = 'Add';
          } else {
            this.loadError = true;
          }
        },
      });
    }
  }

  getErrorMessage(
    formControlName: 'firstName' | 'middleName' | 'lastName' | 'birthdate',
  ) {
    for (const validation of this.validationMessages[formControlName]) {
      if (this.form.get(formControlName)?.hasError(validation.type)) {
        return validation.message;
      }
    }
    return '';
  }

  onSubmit() {
    if (this.form.valid) {
      const userId = Number(this.userId) || 0;
      if (this.type == 'Add') {
        const profile: AddProfile = {
          firstName: this.form.get('firstName')?.value || '',
          middleName: this.form.get('middleName')?.value || '',
          lastName: this.form.get('lastName')?.value || '',
          birthdate: this.form.get('birthdate')?.value || '',
          userId: userId,
        };
        this.profileService.addProfile(profile).subscribe({
          next: (response) => {
            this.router.navigate([`../../user/${response.userId}`], {
              relativeTo: this.route,
            });
          },
          // ProfilesService already surfaces a snackbar on failure; this
          // just keeps a failed submit from going unhandled now that
          // errors are no longer swallowed silently.
          error: () => undefined,
        });
      } else {
        const profile: EditProfile = {
          firstName: this.form.get('firstName')?.value || '',
          middleName: this.form.get('middleName')?.value || '',
          lastName: this.form.get('lastName')?.value || '',
          birthdate: this.form.get('birthdate')?.value || '',
        };
        this.profileService.editProfile(userId, profile).subscribe({
          next: (response) => {
            this.router.navigate([`../../user/${response.userId}`], {
              relativeTo: this.route,
            });
          },
          error: () => undefined,
        });
      }
    }
  }

  onReset() {
    this.form.reset({
      firstName: this.profile?.firstName,
      middleName: this.profile?.middleName || '',
      lastName: this.profile?.lastName,
      birthdate: this.profile?.birthdate,
    });
  }
}
