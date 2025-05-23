import {
  ColDef,
  GridOptions,
  GridReadyEvent,
  RowSelectionOptions,
} from 'ag-grid-community';
import { AgGridAngular } from 'ag-grid-angular';
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
import { CommonModule } from '@angular/common';
import { MatFormField } from '@angular/material/form-field';
import {
  MatAutocomplete,
  MatAutocompleteTrigger,
  MatOption,
} from '@angular/material/autocomplete';
import { MatError, MatInput, MatLabel } from '@angular/material/input';
import { MatButton } from '@angular/material/button';
import { Component, inject, OnInit, ViewChild } from '@angular/core';
import { RolesService, UsersService } from '@jdw/angular-shared-data-access';
import { forkJoin, map, Observable, startWith } from 'rxjs';

type UserRoleAssignmentFormControls = {
  user: FormControl<User | null>;
  rolesToAdd: FormControl<number[]>;
  rolesToRemove: FormControl<number[]>;
};

type FormControlName = keyof UserRoleAssignmentFormControls | 'form';

@Component({
  selector: 'lib-user-role-assignment',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormField,
    MatAutocompleteTrigger,
    MatInput,
    MatAutocomplete,
    MatOption,
    MatButton,
    FormsModule,
    MatError,
    MatLabel,
    AgGridAngular,
  ],
  templateUrl: './user-role-assignment.component.html',
  styleUrl: './user-role-assignment.component.scss',
})
export class UserRoleAssignmentComponent implements OnInit {
  @ViewChild(AgGridAngular) grid!: AgGridAngular;

  private usersService: UsersService = inject(UsersService);
  private rolesService: RolesService = inject(RolesService);

  private noRoleChangeValidator: ValidatorFn = (
    group: AbstractControl,
  ): ValidationErrors | null => {
    const toAdd = group.get('rolesToAdd')?.value as number[] | undefined;
    const toRemove = group.get('rolesToRemove')?.value as number[] | undefined;

    if (!toAdd || !toRemove) return null;

    // Check “no change”: both empty
    const bothEmpty = toAdd.length === 0 && toRemove.length === 0;

    const sameSets =
      toAdd.length === toRemove.length &&
      toAdd.every((id) => toRemove.includes(id));

    return bothEmpty || sameSets ? { noRoleChange: true } : null;
  };

  form: FormGroup<UserRoleAssignmentFormControls> = new FormGroup(
    {
      user: new FormControl<User | null>(null, {
        nonNullable: true,
        validators: [Validators.required, this.invalidUserValidator.bind(this)],
      }),
      rolesToAdd: new FormControl<number[]>([], {
        nonNullable: true,
      }),
      rolesToRemove: new FormControl<number[]>([], {
        nonNullable: true,
      }),
    },
    {
      validators: [this.noRoleChangeValidator],
    },
  );

  validationMessages: Record<
    FormControlName,
    {
      type: string;
      message: string;
    }[]
  > = {
    user: [
      { type: 'required', message: 'User is required' },
      { type: 'invalidUser', message: 'Please select a user from the list' },
    ],
    rolesToAdd: [],
    rolesToRemove: [],
    form: [
      {
        type: 'noRoleChange',
        message: 'You must add or remove at least one role.',
      },
    ],
  };

  allRoles: Role[] = [];
  allUsers: User[] = [];
  currentUserRoles: Role[] = [];
  filteredOptions!: Observable<User[]>;
  loading = true;

  colDefs: ColDef[] = [
    { field: 'name', headerName: 'Name' },
    { field: 'description', headerName: 'Description' },
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
    this.setupUserAutocomplete();
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

  private setupUserAutocomplete(): void {
    this.filteredOptions = this.userControl.valueChanges.pipe(
      startWith(''),
      map((value) => this.filterUsers(value)),
    );
  }

  private filterUsers(value: string | User | null): User[] {
    const filter = typeof value === 'string' ? value.toLowerCase() : '';
    return this.allUsers.filter((user) =>
      user.emailAddress.toLowerCase().includes(filter),
    );
  }

  // === Form Control Accessors ===

  get userControl(): FormControl<User | null> {
    return this.form.get('user') as FormControl<User | null>;
  }

  // === Autocomplete Display ===

  displayUserFn(user: User): string {
    return user ? `${user.emailAddress} (${user.id})` : '';
  }

  // === Grid Logic ===

  onGridReady(_: GridReadyEvent): void {
    this.updateGridSelection();
  }

  private updateGridSelection(): void {
    if (!this.grid.api || !this.currentUserRoles.length) return;

    this.grid.api.forEachNode((node) => {
      const selected = this.currentUserRoles.some((r) => r.id === node.data.id);
      node.setSelected(selected);
    });

    this.onRoleSelectionChanged();
  }

  onUserSelected(user: User): void {
    if (!user) return;

    this.loading = true;

    this.usersService.getUser(user.id).subscribe({
      next: (freshUser) => {
        this.userControl.setValue(freshUser);

        this.currentUserRoles = this.allRoles.filter((role) =>
          freshUser.roles.some((r) => r.roleId === role.id),
        );

        this.grid?.api?.deselectAll();
        this.updateGridSelection();

        this.form.patchValue({
          rolesToAdd: [],
          rolesToRemove: [],
        });

        this.form.markAsTouched();
        this.loading = false;
      },
      error: () => {
        this.loading = false;
      },
    });
  }

  onRoleSelectionChanged(): void {
    const selectedRoles = this.grid.api.getSelectedRows() as Role[];

    const { rolesToAdd, rolesToRemove } = this.calculateRoleDiff(
      selectedRoles,
      this.currentUserRoles,
    );

    this.form.patchValue({ rolesToAdd, rolesToRemove });
  }

  private calculateRoleDiff(
    selected: Role[],
    current: Role[],
  ): { rolesToAdd: number[]; rolesToRemove: number[] } {
    const rolesToAdd = selected
      .filter((r) => !current.some((c) => c.id === r.id))
      .map((r) => r.id);

    const rolesToRemove = current
      .filter((r) => !selected.some((s) => s.id === r.id))
      .map((r) => r.id);

    return { rolesToAdd, rolesToRemove };
  }

  // === Submission & Reset ===

  onSubmit(): void {
    if (!this.form.valid || this.loading) return;

    const { user, rolesToAdd, rolesToRemove } = this.form.value;
    if (!user || !rolesToAdd || !rolesToRemove) return;

    this.loading = true;

    this.usersService
      .assignRolesToUser(user.id, rolesToAdd, rolesToRemove)
      .subscribe({
        next: (updatedUser) => {
          this.currentUserRoles = this.allRoles.filter((role) =>
            updatedUser.roles.some((r) => r.roleId === role.id),
          );
          this.form.patchValue({
            user: null,
            rolesToAdd: [],
            rolesToRemove: [],
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
      user: null,
      rolesToAdd: [],
      rolesToRemove: [],
    });
    this.currentUserRoles = [];

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
    if (this.form.hasError('noRoleChange')) {
      return this.validationMessages['form'][0].message;
    }
    return '';
  }

  private invalidUserValidator(
    control: AbstractControl,
  ): ValidationErrors | null {
    const val = control.value;
    if (val !== null && typeof val === 'string') {
      return { invalidUser: true };
    }
    return null;
  }
}
