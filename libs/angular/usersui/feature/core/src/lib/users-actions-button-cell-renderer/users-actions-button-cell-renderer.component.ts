import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconButton } from '@angular/material/button';
import { MatTooltip } from '@angular/material/tooltip';
import { MatIcon } from '@angular/material/icon';
import { ICellRendererAngularComp } from 'ag-grid-angular';
import { ActivatedRoute, Router } from '@angular/router';
import { ICellRendererParams } from 'ag-grid-community';
import { MatDialog } from '@angular/material/dialog';
import { UserDialogComponent } from '../user-dialog/user-dialog.component';
import { UsersService } from '@jdw/angular-shared-data-access';
import { User } from '@jdw/angular-shared-util';

@Component({
  selector: 'lib-actions-button-cell-renderer',
  imports: [CommonModule, MatIconButton, MatTooltip, MatIcon],
  templateUrl: './users-actions-button-cell-renderer.component.html',
  styleUrl: './users-actions-button-cell-renderer.component.scss',
})
export class UsersActionsButtonCellRendererComponent
  implements ICellRendererAngularComp
{
  private router: Router = inject(Router);
  private route = inject(ActivatedRoute);
  private dialog: MatDialog = inject(MatDialog);
  private params!: ICellRendererParams;
  private usersService: UsersService = inject(UsersService);

  agInit(params: ICellRendererParams): void {
    this.params = params;
  }

  refresh(_: ICellRendererParams): boolean {
    return false;
  }

  viewUser(): void {
    const userId = this.params.data['id'];
    this.router.navigate([`../user/${userId}`], { relativeTo: this.route });
  }

  editUser(): void {
    const userId = this.params.data['id'];
    this.router.navigate([`../account/${userId}`], { relativeTo: this.route });
  }

  deleteUser(): void {
    const dialogRef = this.dialog.open(UserDialogComponent, {
      data: {
        title: 'Delete User',
        type: 'readonly',
        action: 'delete',
        ...this.params.data,
      },
      minWidth: '50%',
    });
    dialogRef.afterClosed().subscribe((result: User) => {
      if (result) {
        this.usersService.deleteUser(result.id).subscribe();
      }
    });
  }
}
