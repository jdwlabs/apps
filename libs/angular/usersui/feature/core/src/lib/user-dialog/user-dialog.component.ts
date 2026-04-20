import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  MAT_DIALOG_DATA,
  MatDialogActions,
  MatDialogContent,
  MatDialogRef,
  MatDialogTitle,
} from '@angular/material/dialog';
import { MatButton } from '@angular/material/button';
import { MatFormField, MatLabel } from '@angular/material/form-field';
import { MatInput } from '@angular/material/input';
import { User } from '@jdw/angular-shared-util';
import {
  FormArray,
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
} from '@angular/forms';
import { UserRole } from '@jdw/angular-shared-util';

export interface DialogData {
  title: string;
  type: 'readonly' | 'readwrite';
  action: string;
}

@Component({
  selector: 'lib-user-dialog',
  imports: [
    CommonModule,
    MatDialogTitle,
    MatDialogContent,
    MatDialogActions,
    MatButton,
    MatFormField,
    MatInput,
    MatLabel,
    ReactiveFormsModule,
  ],
  templateUrl: './user-dialog.component.html',
  styleUrl: './user-dialog.component.scss',
})
export class UserDialogComponent implements OnInit {
  userForm!: FormGroup;
  data: DialogData & User = inject(MAT_DIALOG_DATA);
  private fb = inject(FormBuilder);
  private dialogRef: MatDialogRef<DialogData> = inject(MatDialogRef);

  ngOnInit() {
    this.initializeForm();
    this.populateForm(this.data);
  }

  initializeForm(): void {
    this.userForm = this.fb.group({
      id: [''],
      emailAddress: [''],
      password: [''],
      status: [''],
      createdTime: [''],
      createdByUserId: [''],
      modifiedTime: [''],
      modifiedByUserId: [''],
      roles: this.fb.array([]),
      profile: this.fb.group({
        id: [''],
        firstName: [''],
        middleName: [''],
        lastName: [''],
        birthdate: [''],
        icon: this.fb.group({
          id: [''],
          icon: [''],
          profileId: [''],
          createdTime: [''],
          createdByUserId: [''],
          modifiedTime: [''],
          modifiedByUserId: [''],
        }),
        addresses: this.fb.array([]),
        userId: [''],
        createdTime: [''],
        createdByUserId: [''],
        modifiedTime: [''],
        modifiedByUserId: [''],
      }),
    });
  }

  populateForm(user: User): void {
    this.userForm.patchValue({
      id: user.id,
      emailAddress: user.emailAddress,
      password: user.password,
      status: user.status,
      createdTime: user.createdTime,
      createdByUserId: user.createdByUserId,
      modifiedTime: user.modifiedTime,
      modifiedByUserId: user.modifiedByUserId,
      profile: {
        id: user.profile?.id,
        firstName: user.profile?.firstName,
        middleName: user.profile?.middleName,
        lastName: user.profile?.lastName,
        birthdate: user.profile?.birthdate,
        icon: {
          id: user.profile?.icon?.id,
          icon: user.profile?.icon?.icon,
          profileId: user.profile?.icon?.profileId,
          createdTime: user.profile?.icon?.createdTime,
          createdByUserId: user.profile?.icon?.createdByUserId,
          modifiedTime: user.profile?.icon?.modifiedTime,
          modifiedByUserId: user.profile?.icon?.createdByUserId,
        },
        userId: user.profile?.userId,
        createdTime: user.profile?.createdTime,
        createdByUserId: user.profile?.createdByUserId,
        modifiedTime: user.profile?.modifiedTime,
        modifiedByUserId: user.profile?.modifiedByUserId,
      },
    });

    this.setRoles(user.roles);
    this.setAddresses(user.profile?.addresses ?? []);
  }

  get roles(): FormArray {
    return this.userForm.get('roles') as FormArray;
  }

  get icon(): FormGroup {
    return this.userForm.get('profile')?.get('icon') as FormGroup;
  }

  get addresses(): FormArray {
    return this.userForm.get('profile')?.get('addresses') as FormArray;
  }

  setRoles(roles: UserRole[]) {
    const roleArray = this.userForm.get('roles') as FormArray;
    roles.forEach((role) => {
      roleArray.push(
        this.fb.group({
          userId: [role.userId],
          roleId: [role.roleId],
          createdTime: [role.createdTime],
          createdByUserId: [role.createdByUserId],
        }),
      );
    });
  }

  setAddresses(addresses: any[]) {
    const addressArray = this.userForm
      .get('profile')
      ?.get('addresses') as FormArray;
    addresses.forEach((address) => {
      addressArray.push(
        this.fb.group({
          id: [address.id],
          addressLine1: [address.addressLine1],
          addressLine2: [address.addressLine2],
          city: [address.city],
          stateProvince: [address.stateProvince],
          postalCode: [address.postalCode],
          country: [address.country],
          profileId: [address.profileId],
          createdTime: [address.createdTime],
          createdByUserId: [address.createdByUserId],
          modifiedTime: [address.modifiedTime],
          modifiedByUserId: [address.modifiedByUserId],
        }),
      );
    });
  }

  close(): void {
    this.dialogRef.close();
  }

  action(): void {
    this.dialogRef.close(this.data);
  }
}
