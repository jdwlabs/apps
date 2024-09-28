import { inject, Injectable } from '@angular/core';
import {
  MatSnackBar,
  MatSnackBarConfig,
  MatSnackBarHorizontalPosition,
  MatSnackBarVerticalPosition,
} from '@angular/material/snack-bar';
import { PaletteColors, SnackbarOptions } from '@jdw/angular-shared-util';
/* eslint-disable @nx/enforce-module-boundaries */
import { SnackbarComponent } from '@jdw/angular-shared-ui';
/* eslint-enable @nx/enforce-module-boundaries */

@Injectable({
  providedIn: 'root',
})
export class SnackbarService {
  private duration = 3000;
  private horizontalPosition: MatSnackBarHorizontalPosition = 'end';
  private verticalPosition: MatSnackBarVerticalPosition = 'bottom';
  private snackbar = inject(MatSnackBar);

  send(type: PaletteColors, message: string, options?: SnackbarOptions) {
    const panelClass = [];
    if (!options?.buttonText) panelClass.push('icon');
    if (options?.variant) panelClass.push(options.variant);

    const config: MatSnackBarConfig = {
      data: {
        message: message,
        icon: options?.icon,
        buttonText: options?.buttonText,
      },
      direction: options?.direction,
      duration: options?.autoClose
        ? options?.autoCloseTimeout
          ? options.autoCloseTimeout
          : this.duration
        : 0,
      horizontalPosition:
        options?.horizontalPosition || this.horizontalPosition,
      verticalPosition: options?.verticalPosition || this.verticalPosition,
      panelClass: [...panelClass, type],
    };

    this.snackbar.openFromComponent(SnackbarComponent, config);
  }

  success(message: string, options?: SnackbarOptions, defaultIcon = false) {
    if (defaultIcon) options = { ...options, icon: 'check_circle' };
    this.send('success', message, options);
  }

  error(message: string, options?: SnackbarOptions, defaultIcon = false) {
    if (defaultIcon) options = { ...options, icon: 'report' };
    this.send('error', message, options);
  }

  warn(message: string, options?: SnackbarOptions, defaultIcon = false) {
    if (defaultIcon) options = { ...options, icon: 'warning' };
    this.send('warn', message, options);
  }

  info(message: string, options?: SnackbarOptions, defaultIcon = false) {
    if (defaultIcon) options = { ...options, icon: 'info' };
    this.send('info', message, options);
  }

  clear() {
    this.snackbar.dismiss();
  }
}
