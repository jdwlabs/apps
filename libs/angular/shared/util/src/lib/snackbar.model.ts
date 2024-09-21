import {
  MatSnackBarHorizontalPosition,
  MatSnackBarVerticalPosition,
} from '@angular/material/snack-bar';
import { Direction } from '@angular/cdk/bidi';
import { AlertVariants } from './alert.model';

export type SnackbarData = {
  message: string;
  icon?: string;
  buttonText?: string;
};
export type SnackbarOptions = {
  variant?: AlertVariants;
  autoClose?: boolean;
  autoCloseTimeout?: number;
  icon?: string;
  buttonText?: string;
  direction?: Direction;
  horizontalPosition?: MatSnackBarHorizontalPosition;
  verticalPosition?: MatSnackBarVerticalPosition;
};
