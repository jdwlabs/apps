import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { MatIconButton } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { MatIcon } from '@angular/material/icon';
import { MatTooltip } from '@angular/material/tooltip';
import { ThemeDialogComponent } from './theme-dialog.component';

@Component({
  selector: 'jdw-theme-switcher',
  imports: [MatIconButton, MatIcon, MatTooltip],
  templateUrl: './theme-switcher.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ThemeSwitcherComponent {
  private readonly dialog = inject(MatDialog);

  // A dialog rather than a menu: MatMenu only registers MatMenuItem elements
  // with its FocusKeyManager, so a raw colour <input> inside it is unreachable
  // by keyboard, and Tab on a menu item closes the menu before the input can
  // be tabbed to. A dialog traps focus natively, so every control is reachable.
  open(): void {
    this.dialog.open(ThemeDialogComponent);
  }
}
