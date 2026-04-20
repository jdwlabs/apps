import { Component, inject, OnInit } from '@angular/core';
import {
  FormArray,
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
} from '@angular/forms';
import { Profile } from '@jdw/angular-shared-util';
import {
  MAT_DIALOG_DATA,
  MatDialogActions,
  MatDialogContent,
  MatDialogRef,
  MatDialogTitle,
} from '@angular/material/dialog';
import { CommonModule } from '@angular/common';
import { MatButton } from '@angular/material/button';
import { MatFormField, MatLabel } from '@angular/material/form-field';
import { MatInput } from '@angular/material/input';

export interface DialogData {
  title: string;
  type: 'readonly' | 'readwrite';
  action: string;
}

@Component({
  selector: 'lib-profile-dialog',
  imports: [
    CommonModule,
    MatButton,
    MatDialogActions,
    MatDialogContent,
    MatDialogTitle,
    MatFormField,
    MatInput,
    MatLabel,
    ReactiveFormsModule,
  ],
  templateUrl: './profile-dialog.component.html',
  styleUrl: './profile-dialog.component.scss',
})
export class ProfileDialogComponent implements OnInit {
  profileForm!: FormGroup;
  data: DialogData & Profile = inject(MAT_DIALOG_DATA);
  private fb = inject(FormBuilder);
  private dialogRef: MatDialogRef<DialogData> = inject(MatDialogRef);

  ngOnInit() {
    this.initializeForm();
    this.populateForm(this.data);
  }

  initializeForm(): void {
    this.profileForm = this.fb.group({
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
    });
  }

  populateForm(profile: Profile): void {
    this.profileForm.patchValue({
      id: profile?.id,
      firstName: profile?.firstName,
      middleName: profile?.middleName,
      lastName: profile?.lastName,
      birthdate: profile?.birthdate,
      icon: {
        id: profile?.icon?.id,
        icon: profile?.icon?.icon,
        profileId: profile?.icon?.id,
        createdTime: profile?.icon?.createdTime,
        createdByUserId: profile?.icon?.createdByUserId,
        modifiedTime: profile?.icon?.modifiedTime,
        modifiedByUserId: profile?.icon?.createdByUserId,
      },
      userId: profile?.userId,
      createdTime: profile?.createdTime,
      createdByUserId: profile?.createdByUserId,
      modifiedTime: profile?.modifiedTime,
      modifiedByUserId: profile?.modifiedByUserId,
    });
    this.setAddresses(profile.addresses ?? []);
  }

  get icon(): FormGroup {
    return this.profileForm.get('icon') as FormGroup;
  }

  get addresses(): FormArray {
    return this.profileForm.get('addresses') as FormArray;
  }

  setAddresses(addresses: any[]) {
    const addressArray = this.profileForm.get('addresses') as FormArray;
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
