import { TestBed } from '@angular/core/testing';

import { SnackbarService } from './snackbar.service';
import {
  MatSnackBar,
  MatSnackBarHorizontalPosition,
  MatSnackBarVerticalPosition,
} from '@angular/material/snack-bar';
import { PaletteColors, SnackbarOptions } from '@jdw/angular-shared-util';
/* eslint-disable @nx/enforce-module-boundaries */
import { SnackbarComponent } from '@jdw/angular-shared-ui';
/* eslint-enable @nx/enforce-module-boundaries */

describe('SnackbarService', () => {
  let service: SnackbarService;

  const snackbarMock = {
    openFromComponent: jest.fn(),
    dismiss: jest.fn(),
  };
  const defaultType: PaletteColors = 'primary';
  const defaultMessage = 'test message';
  const defaultDuration = 3000;
  const defaultHorizontalPosition: MatSnackBarHorizontalPosition = 'end';
  const defaultVerticalPosition: MatSnackBarVerticalPosition = 'bottom';

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        {
          provide: MatSnackBar,
          useValue: snackbarMock,
        },
      ],
    });
    service = TestBed.inject(SnackbarService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should create an instance of SnackbarService', () => {
    expect(service).toBeInstanceOf(SnackbarService);
  });

  it('should send a snackbar', () => {
    service.send(defaultType, defaultMessage);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: undefined,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', defaultType],
      },
    );
  });

  it('should send a snackbar with all options', () => {
    const options: SnackbarOptions = {
      variant: 'filled',
      autoClose: true,
      autoCloseTimeout: 5000,
      icon: 'bug_report',
      buttonText: 'Test',
      direction: 'rtl',
      horizontalPosition: 'start',
      verticalPosition: 'top',
    };

    service.send(defaultType, defaultMessage, options);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: options.icon,
          buttonText: options.buttonText,
        },
        direction: options.direction,
        duration: options.autoCloseTimeout,
        horizontalPosition: options.horizontalPosition,
        verticalPosition: options.verticalPosition,
        panelClass: [options.variant, defaultType],
      },
    );
  });

  it('should send a snackbar with some options', () => {
    const options: SnackbarOptions = {
      variant: 'default',
      autoClose: true,
    };

    service.send(defaultType, defaultMessage, options);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: undefined,
          buttonText: undefined,
        },
        direction: undefined,
        duration: defaultDuration,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', options.variant, defaultType],
      },
    );
  });

  it('should send a success snackbar', () => {
    service.success(defaultMessage);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: undefined,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'success'],
      },
    );
  });

  it('should send a success snackbar with defaultIcon', () => {
    service.success(defaultMessage, undefined, true);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: 'check_circle',
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'success'],
      },
    );
  });

  it('should send a success snackbar with options', () => {
    const options: SnackbarOptions = {
      variant: 'outlined',
      icon: 'bug_report',
    };
    service.success(defaultMessage, options, false);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: options.icon,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', options.variant, 'success'],
      },
    );
  });

  it('should send a success snackbar with default icon overriding options', () => {
    const options: SnackbarOptions = {
      icon: 'bug_report',
    };
    service.success(defaultMessage, options, true);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: 'check_circle',
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'success'],
      },
    );
  });

  it('should send an error snackbar', () => {
    service.error(defaultMessage);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: undefined,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'error'],
      },
    );
  });

  it('should send an error snackbar with defaultIcon', () => {
    service.error(defaultMessage, undefined, true);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: 'report',
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'error'],
      },
    );
  });

  it('should send an error snackbar with options', () => {
    const options: SnackbarOptions = {
      variant: 'outlined',
      icon: 'bug_report',
    };
    service.error(defaultMessage, options, false);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: options.icon,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', options.variant, 'error'],
      },
    );
  });

  it('should send an error snackbar with default icon overriding options', () => {
    const options: SnackbarOptions = {
      icon: 'bug_report',
    };
    service.error(defaultMessage, options, true);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: 'report',
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'error'],
      },
    );
  });

  it('should send a warn snackbar', () => {
    service.warn(defaultMessage);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: undefined,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'warn'],
      },
    );
  });

  it('should send a warn snackbar with defaultIcon', () => {
    service.warn(defaultMessage, undefined, true);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: 'warning',
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'warn'],
      },
    );
  });

  it('should send a warn snackbar with options', () => {
    const options: SnackbarOptions = {
      variant: 'outlined',
      icon: 'bug_report',
    };
    service.warn(defaultMessage, options, false);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: options.icon,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', options.variant, 'warn'],
      },
    );
  });

  it('should send a warn snackbar with default icon overriding options', () => {
    const options: SnackbarOptions = {
      icon: 'bug_report',
    };
    service.warn(defaultMessage, options, true);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: 'warning',
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'warn'],
      },
    );
  });

  it('should send an info snackbar', () => {
    service.info(defaultMessage);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: undefined,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'info'],
      },
    );
  });

  it('should send an info snackbar with defaultIcon', () => {
    service.info(defaultMessage, undefined, true);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: 'info',
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'info'],
      },
    );
  });

  it('should send an info snackbar with options', () => {
    const options: SnackbarOptions = {
      variant: 'outlined',
      icon: 'bug_report',
    };
    service.info(defaultMessage, options, false);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: options.icon,
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', options.variant, 'info'],
      },
    );
  });

  it('should send an info snackbar with default icon overriding options', () => {
    const options: SnackbarOptions = {
      icon: 'bug_report',
    };
    service.info(defaultMessage, options, true);

    expect(snackbarMock.openFromComponent).toHaveBeenCalledWith(
      SnackbarComponent,
      {
        data: {
          message: defaultMessage,
          icon: 'info',
          buttonText: undefined,
        },
        direction: undefined,
        duration: 0,
        horizontalPosition: defaultHorizontalPosition,
        verticalPosition: defaultVerticalPosition,
        panelClass: ['icon', 'info'],
      },
    );
  });

  it('should clear the snackbar', () => {
    service.clear();

    expect(snackbarMock.dismiss).toHaveBeenCalled();
  });
});
