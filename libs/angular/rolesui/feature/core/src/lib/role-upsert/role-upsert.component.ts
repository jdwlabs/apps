import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { MatFormField, MatInput } from '@angular/material/input';
import {
  DESCRIPTION_REQUIRED_VALIDATION_MESSAGE,
  NAME_REQUIRED_VALIDATION_MESSAGE,
} from '@jdw/util';
import { ActivatedRoute, Router } from '@angular/router';
import { Role } from '@jdw/angular-shared-util';
import { RolesService } from '@jdw/angular-shared-data-access';
import { finalize } from 'rxjs';
import { MatError, MatLabel } from '@angular/material/form-field';
import { MatButton } from '@angular/material/button';

@Component({
  selector: 'lib-role-upsert',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormField,
    MatInput,
    MatLabel,
    MatError,
    MatFormField,
    MatButton,
  ],
  templateUrl: './role-upsert.component.html',
  styleUrl: './role-upsert.component.scss',
})
export class RoleUpsertComponent implements OnInit {
  type = 'Add';
  form: FormGroup = new FormGroup({
    name: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
    description: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
  });
  validationMessages = {
    name: [{ type: 'required', message: NAME_REQUIRED_VALIDATION_MESSAGE }],
    description: [
      { type: 'required', message: DESCRIPTION_REQUIRED_VALIDATION_MESSAGE },
    ],
  };
  roleId: string | null = null;
  role: Role | null = null;
  private router: Router = inject(Router);
  private rolesService: RolesService = inject(RolesService);
  private route: ActivatedRoute = inject(ActivatedRoute);

  ngOnInit(): void {
    this.roleId = this.route.snapshot.paramMap.get('roleId');

    if (this.roleId) {
      this.rolesService
        .getRole(this.roleId)
        .pipe(
          finalize(() => {
            if (!this.roleId) {
              this.type = 'Add';
              this.router.navigate(['../../'], {
                relativeTo: this.route,
              });
            }
          }),
        )
        .subscribe({
          next: (response) => {
            if (response) {
              this.role = response;
              this.type = 'Edit';
              this.form.patchValue({
                name: response.name,
                description: response.description,
              });
            }
          },
        });
    }
  }

  getErrorMessage(formControlName: 'name' | 'description'): string {
    for (const validation of this.validationMessages[formControlName]) {
      if (this.form.get(formControlName)?.hasError(validation.type)) {
        return validation.message;
      }
    }
    return '';
  }

  onSubmit(): void {
    if (this.form.valid) {
      const roleId = this.roleId || 0;
      const role = {
        name: this.form.get('name')?.value || '',
        description: this.form.get('description')?.value || '',
      };
      if (this.type == 'Add') {
        this.rolesService.addRole(role).subscribe({
          next: (response) => {
            this.router.navigate([`./${response.id}`], {
              relativeTo: this.route,
            });
          },
        });
      } else {
        this.rolesService.editRole(roleId, role).subscribe({
          next: () => {
            this.router.navigate([`../`], {
              relativeTo: this.route,
            });
          },
        });
      }
    }
  }

  onReset(): void {
    this.form.reset({
      name: this.role?.name,
      description: this.role?.description,
    });
  }
}
