import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ICellRendererAngularComp } from 'ag-grid-angular';
import { ActivatedRoute, Router } from '@angular/router';
import { ICellRendererParams } from 'ag-grid-community';
import { MatIcon } from '@angular/material/icon';
import { MatIconButton } from '@angular/material/button';
import { MatTooltip } from '@angular/material/tooltip';

@Component({
  selector: 'lib-profiles-action-button-cell-renderer',
  standalone: true,
  imports: [CommonModule, MatIcon, MatIconButton, MatTooltip],
  templateUrl: './profiles-action-button-cell-renderer.component.html',
  styleUrl: './profiles-action-button-cell-renderer.component.scss',
})
export class ProfilesActionButtonCellRendererComponent
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
    const userId = this.params.data['userId'];
    this.router.navigate([`../user/${userId}`], { relativeTo: this.route });
  }
}
