import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { RolesService } from '@jdw/angular-shared-data-access';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { Role } from '@jdw/angular-shared-util';
import { ConfirmationDialogComponent } from '@jdw/angular-shared-ui';
import { MatButton } from '@angular/material/button';
import { MatIcon } from '@angular/material/icon';
import { MatFormField, MatInput } from '@angular/material/input';
import { MatLabel } from '@angular/material/form-field';

@Component({
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButton,
    MatLabel,
    MatIcon,
    MatFormField,
    MatInput,
    MatFormField,
  ],
  selector: 'lib-role',
  styleUrl: './role.component.scss',
  templateUrl: './role.component.html',
})
export class RoleComponent implements OnInit {
  roleId: string | null = null;
  role: Role | null = null;
  form!: FormGroup;
  private router: Router = inject(Router);
  private route: ActivatedRoute = inject(ActivatedRoute);
  private dialog: MatDialog = inject(MatDialog);
  private rolesService: RolesService = inject(RolesService);
  private fb: FormBuilder = inject(FormBuilder);

  ngOnInit() {
    this.initializeForm();

    this.roleId = this.route.snapshot.paramMap.get('roleId');

    if (this.roleId) {
      this.rolesService.getRole(this.roleId).subscribe({
        next: (response) => {
          this.role = response;
          this.populateForm(this.role);
        },
      });
    }
  }

  initializeForm(): void {
    this.form = this.fb.group({
      name: [''],
      description: [''],
      status: [''],
      usersCount: [''],
    });
  }

  populateForm(role: Role): void {
    this.form.patchValue({
      name: role.name,
      description: role.description,
      status: role.status,
      usersCount: role.users.length,
    });
  }

  deleteRole(): void {
    const dialogRef = this.dialog.open(ConfirmationDialogComponent, {
      data: {
        title: 'Delete Role',
        content: 'Are you sure you want to delete this role?',
        action: 'delete',
      },
      minWidth: '50%',
    });
    dialogRef.afterClosed().subscribe((result) => {
      if (result && this.role) {
        this.rolesService.deleteRole(this.role.id).subscribe(() => {
          this.router.navigate([`../../`], {
            relativeTo: this.route,
          });
        });
      }
    });
  }

  editRole(): void {
    this.router.navigate(['./edit'], {
      relativeTo: this.route,
    });
  }
}
