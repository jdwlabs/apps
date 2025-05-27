import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormArray,
  FormBuilder,
  FormGroup,
  FormsModule,
  ReactiveFormsModule,
} from '@angular/forms';
import {
  MAT_DIALOG_DATA,
  MatDialogActions,
  MatDialogContent,
  MatDialogRef,
  MatDialogTitle,
} from '@angular/material/dialog';
import { Role, UserRole } from '@jdw/angular-shared-util';
import { MatFormField, MatInput } from '@angular/material/input';
import { MatButton } from '@angular/material/button';
import { MatLabel } from '@angular/material/form-field';

export interface DialogData {
  title: string;
  type: 'readonly' | 'readwrite';
  action: string;
}

@Component({
  selector: 'lib-roles-dialog',
  imports: [
    CommonModule,
    FormsModule,
    MatButton,
    MatDialogActions,
    MatDialogContent,
    MatDialogTitle,
    MatFormField,
    MatLabel,
    MatInput,
    MatFormField,
    ReactiveFormsModule,
  ],
  templateUrl: './roles-dialog.component.html',
  styleUrl: './roles-dialog.component.scss',
})
export class RolesDialogComponent implements OnInit {
  form!: FormGroup;
  data: DialogData & Role = inject(MAT_DIALOG_DATA);
  private fb: FormBuilder = inject(FormBuilder);
  private dialogRef: MatDialogRef<DialogData> = inject(MatDialogRef);

  ngOnInit(): void {
    this.initializeForm();
    this.populateForm(this.data);
  }

  initializeForm(): void {
    this.form = this.fb.group({
      id: [''],
      name: [''],
      description: [''],
      status: [''],
      users: this.fb.array([]),
      createdByUserId: [''],
      createdTime: [''],
      modifiedByUserId: [''],
      modifiedTime: [''],
    });
  }

  populateForm(role: Role): void {
    this.form.patchValue({
      id: role.id,
      name: role.name,
      description: role.description,
      status: role.status,
      createdByUserId: role.createdByUserId,
      createdTime: role.createdTime,
      modifiedByUserId: role.createdByUserId,
      modifiedTime: role.createdTime,
    });

    this.setUsers(role.users || []);
  }

  setUsers(users: UserRole[]): void {
    const usersArray = this.form.get('users') as FormArray;
    users.forEach((user) => {
      usersArray.push(
        this.fb.group({
          userId: [user.userId],
          roleId: [user.roleId],
          createdTime: [user.createdTime],
          createdByUserId: [user.createdByUserId],
        }),
      );
    });
  }

  get users(): FormArray {
    return this.form.get('users') as FormArray;
  }

  close(): void {
    this.dialogRef.close();
  }

  action(): void {
    this.dialogRef.close(this.data);
  }
}
