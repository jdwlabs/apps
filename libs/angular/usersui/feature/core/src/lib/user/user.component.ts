import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { User } from '@jdw/angular-usersui-util';
import { ActivatedRoute } from '@angular/router';
import { UsersService } from '@jdw/angular-usersui-data-access';
import {
  FormArray,
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
} from '@angular/forms';
import { MatError, MatFormField, MatLabel } from '@angular/material/form-field';
import { MatInput } from '@angular/material/input';
import { MatOption } from '@angular/material/autocomplete';
import { MatSelect } from '@angular/material/select';
import { MatButton } from '@angular/material/button';
import {
  MatCard,
  MatCardContent,
  MatCardHeader,
  MatCardImage,
  MatCardTitle,
} from '@angular/material/card';
import { MatChip, MatChipSet } from '@angular/material/chips';
import { RolesService } from '@jdw/angular-shared-data-access';

@Component({
  selector: 'lib-user',
  standalone: true,
  imports: [
    CommonModule,
    MatFormField,
    ReactiveFormsModule,
    MatInput,
    MatOption,
    MatSelect,
    MatButton,
    MatLabel,
    MatError,
    MatCard,
    MatCardHeader,
    MatCardContent,
    MatCardImage,
    MatCardTitle,
    MatChip,
    MatChipSet,
  ],
  templateUrl: './user.component.html',
  styleUrl: './user.component.scss',
})
export class UserComponent implements OnInit {
  userId: string | null = null;
  user: User | null = null;
  userForm!: FormGroup;
  private route = inject(ActivatedRoute);
  private usersService = inject(UsersService);
  private fb = inject(FormBuilder);
  private rolesService = inject(RolesService);

  ngOnInit(): void {
    this.initializeForm();

    this.userId = this.route.snapshot.paramMap.get('userId');

    if (this.userId) {
      this.usersService.getUser(this.userId).subscribe({
        next: (response) => {
          this.user = response;
          this.populateForm(this.user);
        },
      });
    }
  }

  get roles(): FormArray {
    return this.userForm.get('roles') as FormArray;
  }

  get addresses(): FormArray {
    return this.userForm.get('profile')?.get('addresses') as FormArray;
  }

  initializeForm() {
    this.userForm = this.fb.group({
      emailAddress: [''],
      password: [''],
      status: [''],
      roles: this.fb.array([]),
      profile: this.fb.group({
        firstName: [''],
        middleName: [''],
        lastName: [''],
        birthdate: [''],
        icon: [''],
        addresses: this.fb.array([]),
      }),
    });
  }

  populateForm(user: User) {
    this.userForm.patchValue({
      emailAddress: user.emailAddress,
      password: user.password,
      status: user.status,
      profile: {
        firstName: user.profile?.firstName,
        middleName: user.profile?.middleName,
        lastName: user.profile?.lastName,
        birthdate: user.profile?.birthdate,
        icon: user.profile?.icon?.icon,
      },
    });

    this.setRoles(user.roles);
    this.setAddresses(user.profile?.addresses ?? []);
  }

  setRoles(roles: any[]) {
    const roleArray = this.userForm.get('roles') as FormArray;
    roles.forEach((role) => {
      this.rolesService.getRole(role.roleId).subscribe({
        next: (roleResponse) => {
          roleArray.push(this.fb.control(roleResponse.name));
        },
      });
    });
  }

  setAddresses(addresses: any[]) {
    const addressArray = this.userForm
      .get('profile')
      ?.get('addresses') as FormArray;
    addresses.forEach((address) => {
      addressArray.push(
        this.fb.group({
          addressLine1: [address.addressLine1],
          addressLine2: [address.addressLine2],
          city: [address.city],
          stateProvince: [address.stateProvince],
          postalCode: [address.postalCode],
          country: [address.country],
        }),
      );
    });
  }
}
