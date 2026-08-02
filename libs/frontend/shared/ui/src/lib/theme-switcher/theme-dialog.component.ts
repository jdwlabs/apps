import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
} from '@angular/core';
import { MatDialogContent, MatDialogTitle } from '@angular/material/dialog';
import { MatRadioButton, MatRadioGroup } from '@angular/material/radio';
import { ThemeId, ThemeService } from '@jdw/frontend-shared-util';

@Component({
  selector: 'jdw-theme-dialog',
  imports: [MatDialogContent, MatDialogTitle, MatRadioButton, MatRadioGroup],
  templateUrl: './theme-dialog.component.html',
  styleUrl: './theme-dialog.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ThemeDialogComponent {
  private readonly themes = inject(ThemeService);

  readonly options = this.themes.themes;
  readonly current = this.themes.themeId;
  readonly colour = this.themes.customColour;

  readonly customisable = computed(
    () =>
      this.options.find((theme) => theme.id === this.current())?.customisable ??
      false,
  );

  choose(id: ThemeId): void {
    this.themes.select(id);
  }

  recolour(hex: string): void {
    this.themes.setCustomColour(hex);
  }
}
