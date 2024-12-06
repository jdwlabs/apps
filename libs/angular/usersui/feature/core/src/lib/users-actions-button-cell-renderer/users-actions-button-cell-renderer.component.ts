import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconButton } from '@angular/material/button';
import { MatTooltip } from '@angular/material/tooltip';
import { MatIcon } from '@angular/material/icon';
import { ICellRendererAngularComp } from 'ag-grid-angular';
import { ActivatedRoute, Router } from '@angular/router';
import { ICellRendererParams } from 'ag-grid-community';

@Component({
  selector: 'lib-actions-button-cell-renderer',
  standalone: true,
  imports: [CommonModule, MatIconButton, MatTooltip, MatIcon],
  templateUrl: './users-actions-button-cell-renderer.component.html',
  styleUrl: './users-actions-button-cell-renderer.component.scss',
})
export class UsersActionsButtonCellRendererComponent
  implements ICellRendererAngularComp
{
  private router: Router = inject(Router);
  private route = inject(ActivatedRoute);
  private params!: ICellRendererParams;

  agInit(params: ICellRendererParams): void {
    this.params = params;
  }

  refresh(params: ICellRendererParams): boolean {
    return false;
  }

  viewUser(): void {
    const userId = this.params.data['id'];
    this.router.navigate([`../user/${userId}`], { relativeTo: this.route });
  }
}
