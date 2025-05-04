import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ICellRendererAngularComp } from 'ag-grid-angular';
import { ICellRendererParams } from 'ag-grid-community';
import { MatDialog } from '@angular/material/dialog';
import { ActivatedRoute, Router } from '@angular/router';
import { RolesService } from '@jdw/angular-shared-data-access';
import { MatIcon } from '@angular/material/icon';
import { MatIconButton } from '@angular/material/button';
import { MatTooltip } from '@angular/material/tooltip';
import { RolesDialogComponent } from '../roles-dialog/roles-dialog.component';

@Component({
  selector: 'lib-roles-action-button-cell-renderer',
  imports: [CommonModule, MatIcon, MatIconButton, MatTooltip],
  templateUrl: './roles-action-button-cell-renderer.component.html',
  styleUrl: './roles-action-button-cell-renderer.component.scss',
})
export class RolesActionButtonCellRendererComponent
  implements ICellRendererAngularComp
{
  private router: Router = inject(Router);
  private route: ActivatedRoute = inject(ActivatedRoute);
  private dialog: MatDialog = inject(MatDialog);
  private params!: ICellRendererParams;
  private rolesService: RolesService = inject(RolesService);

  agInit(params: ICellRendererParams): void {
    this.params = params;
  }

  refresh(_: ICellRendererParams): boolean {
    return false;
  }

  viewRole(): void {
    const roleId = this.params.data['id'];
    this.router.navigate([`../role/${roleId}`], { relativeTo: this.route });
  }

  editRole(): void {
    const roleId = this.params.data['id'];
    this.router.navigate([`../role/${roleId}/edit`], {
      relativeTo: this.route,
    });
  }

  deleteRole(): void {
    const dialogRef = this.dialog.open(RolesDialogComponent, {
      data: {
        title: 'Delete Role',
        type: 'readonly',
        action: 'delete',
        ...this.params.data,
      },
      minWidth: '50%',
    });
    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        this.rolesService.deleteRole(result.roleId).subscribe();
      }
    });
  }
}
