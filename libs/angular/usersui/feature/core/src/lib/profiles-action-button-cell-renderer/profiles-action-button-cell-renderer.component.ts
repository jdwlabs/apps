import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ICellRendererAngularComp } from 'ag-grid-angular';
import { ActivatedRoute, Router } from '@angular/router';
import { ICellRendererParams } from 'ag-grid-community';
import { MatIcon } from '@angular/material/icon';
import { MatIconButton } from '@angular/material/button';
import { MatTooltip } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import { ProfileDialogComponent } from '../profile-dialog/profile-dialog.component';

@Component({
  selector: 'lib-profiles-action-button-cell-renderer',
  imports: [CommonModule, MatIcon, MatIconButton, MatTooltip],
  templateUrl: './profiles-action-button-cell-renderer.component.html',
  styleUrl: './profiles-action-button-cell-renderer.component.scss',
})
export class ProfilesActionButtonCellRendererComponent
  implements ICellRendererAngularComp
{
  private router: Router = inject(Router);
  private route = inject(ActivatedRoute);
  private dialog: MatDialog = inject(MatDialog);
  private params!: ICellRendererParams;
  private profilesService: ProfilesService = inject(ProfilesService);

  agInit(params: ICellRendererParams): void {
    this.params = params;
  }

  refresh(_: ICellRendererParams): boolean {
    return false;
  }

  viewUser(): void {
    const userId = this.params.data['userId'];
    this.router.navigate([`../user/${userId}`], { relativeTo: this.route });
  }

  editProfile(): void {
    const userId = this.params.data['userId'];
    this.router.navigate([`../profile/${userId}`], { relativeTo: this.route });
  }

  deleteProfile(): void {
    const dialogRef = this.dialog.open(ProfileDialogComponent, {
      data: {
        title: 'Delete Profile',
        type: 'readonly',
        action: 'delete',
        ...this.params.data,
      },
      minWidth: '50%',
    });
    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        this.profilesService.deleteProfile(result.userId).subscribe();
      }
    });
  }
}
