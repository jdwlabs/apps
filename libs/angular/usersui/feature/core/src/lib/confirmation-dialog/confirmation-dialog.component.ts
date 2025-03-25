import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  MAT_DIALOG_DATA,
  MatDialogActions,
  MatDialogContent,
  MatDialogRef,
  MatDialogTitle,
} from '@angular/material/dialog';
import { MatButton } from '@angular/material/button';

export interface ConfirmationDialogData {
  title: string;
  content: string;
  action: string;
}

@Component({
  selector: 'lib-confirmation-dialog',
  imports: [
    CommonModule,
    MatDialogTitle,
    MatDialogContent,
    MatDialogActions,
    MatButton,
  ],
  templateUrl: './confirmation-dialog.component.html',
  styleUrl: './confirmation-dialog.component.scss',
})
export class ConfirmationDialogComponent {
  protected data: ConfirmationDialogData = inject(MAT_DIALOG_DATA);
  private dialogRef: MatDialogRef<ConfirmationDialogData> =
    inject(MatDialogRef);

  close(): void {
    this.dialogRef.close();
  }

  action(): void {
    this.dialogRef.close(this.data);
  }
}
