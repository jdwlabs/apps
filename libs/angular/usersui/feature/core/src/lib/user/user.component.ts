import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { User } from '@jdw/angular-usersui-util';
import { ActivatedRoute, Router } from '@angular/router';
import {
  ProfilesService,
  UsersService,
} from '@jdw/angular-usersui-data-access';
import {
  FormArray,
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
} from '@angular/forms';
import { MatFormField, MatLabel } from '@angular/material/form-field';
import { MatInput } from '@angular/material/input';
import { MatChip, MatChipSet } from '@angular/material/chips';
import { RolesService } from '@jdw/angular-shared-data-access';
import { MatButton } from '@angular/material/button';
import { MatIcon } from '@angular/material/icon';
import { MatDialog } from '@angular/material/dialog';
import { ConfirmationDialogComponent } from '@jdw/angular-shared-ui';

@Component({
  selector: 'lib-user',
  imports: [
    CommonModule,
    MatFormField,
    ReactiveFormsModule,
    MatInput,
    MatLabel,
    MatChip,
    MatChipSet,
    MatButton,
    MatIcon,
  ],
  templateUrl: './user.component.html',
  styleUrl: './user.component.scss',
  standalone: true,
})
export class UserComponent implements OnInit {
  userId: string | null = null;
  user: User | null = null;
  userForm!: FormGroup;
  private router: Router = inject(Router);
  private route = inject(ActivatedRoute);
  private dialog: MatDialog = inject(MatDialog);
  private usersService = inject(UsersService);
  private fb = inject(FormBuilder);
  private rolesService = inject(RolesService);
  private profilesService = inject(ProfilesService);

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

  deleteUser(): void {
    const dialogRef = this.dialog.open(ConfirmationDialogComponent, {
      data: {
        title: 'Delete User',
        content: 'Are you sure you want to delete this user?',
        action: 'delete',
      },
      minWidth: '50%',
    });
    dialogRef.afterClosed().subscribe((result) => {
      if (result && this.user) {
        this.usersService.deleteUser(this.user.id).subscribe(() => {
          this.router.navigate([`../../`], {
            relativeTo: this.route,
          });
        });
      }
    });
  }

  editProfile(): void {
    this.router.navigate([`../../profile/${this.user?.id}`], {
      relativeTo: this.route,
    });
  }

  deleteProfile(): void {
    const dialogRef = this.dialog.open(ConfirmationDialogComponent, {
      data: {
        title: 'Delete Profile',
        content: 'Are you sure you want to delete this profile?',
        action: 'delete',
      },
      minWidth: '50%',
    });
    dialogRef.afterClosed().subscribe((result) => {
      if (result && this.user) {
        this.profilesService.deleteProfile(this.user.id).subscribe(() => {
          this.reloadPage();
        });
      }
    });
  }

  addAddress(): void {
    this.router.navigate([`../../profile/${this.user?.profile?.id}/address`], {
      relativeTo: this.route,
    });
  }

  editAddress(addressIndex: number): void {
    const addressId = this.user?.profile?.addresses[addressIndex].id;
    this.router.navigate(
      [`../../profile/${this.user?.profile?.id}/address/${addressId}`],
      {
        relativeTo: this.route,
      },
    );
  }

  deleteAddress(addressIndex: number): void {
    const addressId = this.user?.profile?.addresses[addressIndex].id;
    const profileId = this.user?.profile?.id;
    const dialogRef = this.dialog.open(ConfirmationDialogComponent, {
      data: {
        title: 'Delete Address',
        content: 'Are you sure you want to delete this address?',
        action: 'delete',
      },
      minWidth: '50%',
    });
    dialogRef.afterClosed().subscribe((result) => {
      if (result && profileId && addressId) {
        this.profilesService
          .deleteAddress(profileId, addressId)
          .subscribe(() => {
            this.reloadPage();
          });
      }
    });
  }

  deleteIcon(): void {
    const profileId = this.user?.profile?.id;
    const dialogRef = this.dialog.open(ConfirmationDialogComponent, {
      data: {
        title: 'Delete Icon',
        content: 'Are you sure you want to delete this icon?',
        action: 'delete',
      },
      minWidth: '50%',
    });
    dialogRef.afterClosed().subscribe((result) => {
      if (result && profileId) {
        this.profilesService.deleteIcon(profileId).subscribe(() => {
          this.reloadPage();
        });
      }
    });
  }

  editIcon(): void {
    this.router.navigate([`../../profile/${this.user?.profile?.id}/icon`], {
      relativeTo: this.route,
    });
  }

  editAccount(): void {
    this.router.navigate([`../../account/${this.user?.id}`], {
      relativeTo: this.route,
    });
  }

  addProfile(): void {
    this.router.navigate([`../../profile/${this.user?.id}`], {
      relativeTo: this.route,
    });
  }

  addIcon(): void {
    this.router.navigate([`../../profile/${this.user?.profile?.id}/icon`], {
      relativeTo: this.route,
    });
  }

  private reloadPage(): void {
    const currentUrl = this.router.url;
    this.router.navigateByUrl('/', { skipLocationChange: true }).then(() => {
      this.router.navigate([currentUrl]);
    });
  }
}
