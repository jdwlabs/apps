import { Component, inject, ChangeDetectionStrategy } from '@angular/core';

import {
  MAT_SNACK_BAR_DATA,
  MatSnackBarRef,
} from '@angular/material/snack-bar';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'jdw-snackbar',
  imports: [MatIconModule, MatButtonModule],
  templateUrl: './snackbar.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './snackbar.component.scss',
})
export class SnackbarComponent {
  data = inject(MAT_SNACK_BAR_DATA);
  snackbarRef = inject(MatSnackBarRef<SnackbarComponent>);
}
