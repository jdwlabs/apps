import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { Address, AddressRequest } from '@jdw/angular-usersui-util';
import { ActivatedRoute, Router } from '@angular/router';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import { MatButton } from '@angular/material/button';
import { MatError, MatFormField, MatLabel } from '@angular/material/form-field';
import { MatInput } from '@angular/material/input';

@Component({
  selector: 'lib-address',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButton,
    MatError,
    MatFormField,
    MatLabel,
    MatInput,
  ],
  templateUrl: './address.component.html',
  styleUrl: './address.component.scss',
})
export class AddressComponent implements OnInit {
  type = 'Add';
  form = new FormGroup({
    addressLine1: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
    addressLine2: new FormControl('', {
      nonNullable: true,
    }),
    city: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
    stateProvince: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
    postalCode: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
    country: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
  });
  validationMessages = {
    addressLine1: [{ type: 'required', message: 'Address line 1 is required' }],
    addressLine2: [],
    city: [{ type: 'required', message: 'City is required' }],
    stateProvince: [
      { type: 'required', message: 'State / province is required' },
    ],
    postalCode: [{ type: 'required', message: 'Postal code is required' }],
    country: [{ type: 'required', message: 'Country is required' }],
  };
  profileId: string | null = null;
  addressId: string | null = null;
  address: Address | null = null;
  private router: Router = inject(Router);
  private route: ActivatedRoute = inject(ActivatedRoute);
  private profilesService: ProfilesService = inject(ProfilesService);

  ngOnInit() {
    this.profileId = this.route.snapshot.params['profileId'];
    this.addressId = this.route.snapshot.params['addressId'];

    if (this.profileId && this.addressId) {
      this.profilesService
        .getAddress(Number(this.profileId), Number(this.addressId))
        .subscribe({
          next: (data: Address | undefined) => {
            if (data) {
              this.address = data;
              this.type = 'Edit';
              this.form.patchValue({
                addressLine1: data.addressLine1,
                addressLine2: data.addressLine2,
                city: data.city,
                stateProvince: data.stateProvince,
                postalCode: data.postalCode,
                country: data.country,
              });
            } else {
              this.type = 'Add';
              this.router.navigate(['../'], {
                relativeTo: this.route,
              });
            }
          },
        });
    }
  }

  getErrorMessage(formControlName: keyof AddressRequest) {
    for (const validation of this.validationMessages[formControlName]) {
      if (this.form.get(formControlName)?.hasError(validation.type)) {
        return validation.message;
      }
    }
    return '';
  }

  onSubmit() {
    if (this.form.valid) {
      const address: AddressRequest = {
        addressLine1: this.form.get('addressLine1')?.value || '',
        addressLine2: this.form.get('addressLine2')?.value || '',
        city: this.form.get('city')?.value || '',
        stateProvince: this.form.get('stateProvince')?.value || '',
        postalCode: this.form.get('postalCode')?.value || '',
        country: this.form.get('country')?.value || '',
      };

      if (this.type == 'Add') {
        this.profilesService
          .addAddress(Number(this.profileId), address)
          .subscribe({
            next: (response) => {
              this.router.navigate([`../../../user/${response.userId}`], {
                relativeTo: this.route,
              });
            },
          });
      } else {
        this.profilesService
          .editAddress(
            Number(this.profileId),
            Number(this.address?.id),
            address,
          )
          .subscribe({
            next: (response) => {
              this.router.navigate([`../../../../user/${response.userId}`], {
                relativeTo: this.route,
              });
            },
          });
      }
    }
  }

  onReset() {
    this.form.reset({
      addressLine1: this.address?.addressLine1,
      addressLine2: this.address?.addressLine2,
      city: this.address?.city,
      stateProvince: this.address?.stateProvince,
      postalCode: this.address?.postalCode,
      country: this.address?.country,
    });
  }
}
