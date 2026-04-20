import { Component, inject, OnInit, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AgGridAngular } from 'ag-grid-angular';
import {
  MatAutocomplete,
  MatAutocompleteTrigger,
  MatOption,
} from '@angular/material/autocomplete';
import { RolesService, UsersService } from '@jdw/angular-shared-data-access';
import {
  AbstractControl,
  FormControl,
  FormGroup,
  FormsModule,
  ReactiveFormsModule,
  ValidationErrors,
  ValidatorFn,
  Validators,
} from '@angular/forms';
import { Role, User } from '@jdw/angular-shared-util';
import { forkJoin, map, Observable, startWith } from 'rxjs';
import {
  ColDef,
  GridOptions,
  GridReadyEvent,
  RowSelectionOptions,
} from 'ag-grid-community';
import { MatFormField, MatInput } from '@angular/material/input';
import { MatButton } from '@angular/material/button';
import { MatError, MatLabel } from '@angular/material/form-field';

type UserRoleAssignmentFormControls = {
  role: FormControl<Role | null>;
  usersToAdd: FormControl<number[]>;
  usersToRemove: FormControl<number[]>;
};

type FormControlName = keyof UserRoleAssignmentFormControls | 'form';

@Component({
  selector: 'lib-role-user-assignment',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    FormsModule,
    AgGridAngular,
    MatAutocomplete,
    MatAutocompleteTrigger,
    MatInput,
    MatOption,
    MatButton,
    MatError,
    MatLabel,
    MatFormField,
  ],
  templateUrl: './role-user-assignment.component.html',
  styleUrl: './role-user-assignment.component.scss',
})
export class RoleUserAssignmentComponent implements OnInit {
  @ViewChild(AgGridAngular) grid!: AgGridAngular;

  private usersService: UsersService = inject(UsersService);
  private rolesService: RolesService = inject(RolesService);

  private noUserChangeValidator: ValidatorFn = (control: AbstractControl) => {
    const toAdd = control.get('usersToAdd')?.value as number[] | undefined;
    const toRemove = control.get('usersToRemove')?.value as
      | number[]
      | undefined;

    if (!toAdd || !toRemove) return null;

    // Check "no change": both empty
    const bothEmpty = toAdd.length === 0 && toRemove.length === 0;

    const sameSets =
      toAdd.length === toRemove.length &&
      toAdd.every((id) => toRemove.includes(id));

    return bothEmpty || sameSets ? { noUserChange: true } : null;
  };

  form: FormGroup<UserRoleAssignmentFormControls> = new FormGroup(
    {
      role: new FormControl<Role | null>(null, {
        nonNullable: true,
        validators: [Validators.required, this.invalidRoleValidator.bind(this)],
      }),
      usersToAdd: new FormControl<number[]>([], {
        nonNullable: true,
      }),
      usersToRemove: new FormControl<number[]>([], {
        nonNullable: true,
      }),
    },
    {
      validators: [this.noUserChangeValidator],
    },
  );

  validationMessages: Record<
    FormControlName,
    { type: string; message: string }[]
  > = {
    role: [
      { type: 'required', message: 'Role is required' },
      { type: 'invalidRole', message: 'Please select a role from the list' },
    ],
    usersToAdd: [],
    usersToRemove: [],
    form: [
      {
        type: 'noUserChange',
        message: 'You must add or remove at least one user',
      },
    ],
  };

  allUsers: User[] = [];
  allRoles: Role[] = [];
  currentRoleUsers: User[] = [];
  filteredOptions!: Observable<Role[]>;
  loading = true;

  colDefs: ColDef[] = [
    { field: 'emailAddress', headerName: 'Email' },
    { field: 'status', headerName: 'Status' },
  ];

  defaultColDef: ColDef = {
    sortable: true,
    filter: true,
    resizable: true,
    cellStyle: {
      display: 'flex',
      alignItems: 'center',
      whiteSpace: 'nowrap',
      textOverflow: 'ellipsis',
      overflow: 'hidden',
    },
  };

  rowSelection: RowSelectionOptions = {
    mode: 'multiRow',
    headerCheckbox: true,
    checkboxes: true,
    enableClickSelection: 'enableDeselection',
    selectAll: 'filtered',
    enableSelectionWithoutKeys: true,
  };

  gridOptions: GridOptions = {
    columnDefs: this.colDefs,
    defaultColDef: this.defaultColDef,
    pagination: true,
    animateRows: true,
    enableCellTextSelection: true,
    autoSizeStrategy: { type: 'fitCellContents' },
    suppressColumnVirtualisation: true,
  };

  ngOnInit(): void {
    this.loadData();
    this.setupRoleAutocomplete();
  }

  // === Initialization ===

  private loadData(): void {
    forkJoin({
      users: this.usersService.getUsers(),
      roles: this.rolesService.getRoles(),
    }).subscribe(({ users, roles }) => {
      this.allUsers = users;
      this.allRoles = roles;
      this.loading = false;
    });
  }

  private setupRoleAutocomplete(): void {
    this.filteredOptions = this.roleControl.valueChanges.pipe(
      startWith(''),
      map((value) => this.filterRoles(value)),
    );
  }

  private filterRoles(value: string | Role | null): Role[] {
    const filter = typeof value === 'string' ? value.toLowerCase() : '';
    return this.allRoles.filter((role: Role) =>
      role.name.toLowerCase().includes(filter),
    );
  }

  // === Form Control Accessors ===

  get roleControl(): FormControl<Role | null> {
    return this.form.get('role') as FormControl<Role | null>;
  }

  // === Autocomplete Display ===

  displayRoleFn(role: Role): string {
    return role ? `${role.name} (${role.id})` : '';
  }

  // === Grid Logic ===

  onGridReady(_: GridReadyEvent): void {
    this.updateGridSelection();
  }

  private updateGridSelection(): void {
    if (!this.grid.api || !this.currentRoleUsers.length) return;

    const currentRoleIds = new Set(this.currentRoleUsers.map((u) => u.id));

    this.grid.api.forEachNode((node) => {
      const shouldBeSelected = currentRoleIds.has(node.data.id);
      if (node.isSelected() !== shouldBeSelected) {
        node.setSelected(shouldBeSelected);
      }
    });

    this.onUserSelectionChanged();
  }

  onRoleSelected(role: Role): void {
    if (!role) return;

    this.loading = true;

    this.rolesService.getRole(role.id).subscribe({
      next: (freshRole) => {
        this.roleControl.setValue(freshRole);

        this.currentRoleUsers = this.allUsers.filter((user) =>
          freshRole.users.some((u) => u.userId === user.id),
        );

        this.grid?.api?.deselectAll();
        this.updateGridSelection();

        this.form.patchValue({
          usersToAdd: [],
          usersToRemove: [],
        });

        this.form.markAsTouched();
        this.loading = false;
      },
      error: () => {
        this.loading = false;
      },
    });
  }

  onUserSelectionChanged(): void {
    const selectedUsers = this.grid.api.getSelectedRows() as User[];

    const { usersToAdd, usersToRemove } = this.calculateUserDiff(
      selectedUsers,
      this.currentRoleUsers,
    );

    this.form.patchValue({ usersToAdd, usersToRemove });
  }

  private calculateUserDiff(
    selected: User[],
    current: User[],
  ): { usersToAdd: number[]; usersToRemove: number[] } {
    const selectedIds = new Set(selected.map((u) => u.id));
    const currentIds = new Set(current.map((u) => u.id));

    const usersToAdd = [...selectedIds].filter((id) => !currentIds.has(id));
    const usersToRemove = [...currentIds].filter((id) => !selectedIds.has(id));

    return { usersToAdd, usersToRemove };
  }

  // === Submission & Reset ===

  onSubmit(): void {
    if (!this.form.valid || this.loading) return;

    const { role, usersToAdd, usersToRemove } = this.form.value;
    if (!role || !usersToAdd || !usersToRemove) return;

    this.loading = true;

    this.rolesService
      .assignUsersToRole(role.id, usersToAdd, usersToRemove)
      .subscribe({
        next: (updatedRole) => {
          this.currentRoleUsers = this.allUsers.filter((user) =>
            updatedRole.users.some((u) => u.userId === user.id),
          );
          this.form.patchValue({
            role: null,
            usersToAdd: [],
            usersToRemove: [],
          });
          this.updateGridSelection();
          this.form.markAsPristine();
          this.form.markAsUntouched();
          this.grid.api.deselectAll();
          this.loading = false;
        },
        error: () => {
          this.loading = false;
        },
      });
  }

  onReset(): void {
    this.form.reset({
      role: null,
      usersToAdd: [],
      usersToRemove: [],
    });
    this.currentRoleUsers = [];

    if (this.grid && this.grid.api) {
      this.grid.api.deselectAll();
    }
  }

  // === Error Handling ===

  getErrorMessage(controlName: FormControlName): string {
    const control = this.form.get(controlName);
    if (!control?.errors) return '';

    for (const validation of this.validationMessages[controlName]) {
      if (control.hasError(validation.type)) {
        return validation.message;
      }
    }
    return '';
  }

  get formErrorMessage(): string {
    if (this.form.hasError('noUserChange')) {
      return this.validationMessages['form'][0].message;
    }
    return '';
  }

  private invalidRoleValidator(
    control: AbstractControl,
  ): ValidationErrors | null {
    const val = control.value;
    if (val !== null && typeof val === 'string') {
      return { invalidRole: true };
    }
    return null;
  }
}
