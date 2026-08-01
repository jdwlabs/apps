import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
} from '@angular/core';
import { MatIconButton } from '@angular/material/button';
import { MatIcon } from '@angular/material/icon';
import { MatMenu, MatMenuItem, MatMenuTrigger } from '@angular/material/menu';
import { MatTooltip } from '@angular/material/tooltip';
import { ThemeId, ThemeService } from '@jdw/frontend-shared-util';

@Component({
  selector: 'jdw-theme-switcher',
  imports: [
    MatIconButton,
    MatIcon,
    MatMenu,
    MatMenuItem,
    MatMenuTrigger,
    MatTooltip,
  ],
  templateUrl: './theme-switcher.component.html',
  styleUrl: './theme-switcher.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ThemeSwitcherComponent {
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
