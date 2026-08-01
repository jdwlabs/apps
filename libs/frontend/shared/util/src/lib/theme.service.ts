import { DOCUMENT } from '@angular/common';
import { Injectable, inject, signal } from '@angular/core';
import { ToneSet, toneSetFrom } from './on-color.util';
import {
  DEFAULT_CUSTOM_COLOUR,
  DEFAULT_THEME_ID,
  THEMES,
  ThemeDefinition,
  ThemeId,
} from './theme.model';

const THEME_KEY = 'jdw.theme';
const COLOUR_KEY = 'jdw.theme.colour';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  private readonly document = inject(DOCUMENT);

  private readonly selected = signal<ThemeId>(this.restoreTheme());
  private readonly colour = signal<string>(this.restoreColour());

  readonly themeId = this.selected.asReadonly();
  readonly customColour = this.colour.asReadonly();
  readonly themes = THEMES;

  constructor() {
    this.apply();
  }

  select(id: ThemeId): void {
    this.selected.set(id);
    this.persist(THEME_KEY, id);
    this.apply();
  }

  setCustomColour(hex: string): void {
    this.colour.set(hex);
    this.persist(COLOUR_KEY, hex);
    this.apply();
  }

  private definition(): ThemeDefinition {
    return (
      THEMES.find((theme) => theme.id === this.selected()) ??
      THEMES.find((theme) => theme.id === DEFAULT_THEME_ID)!
    );
  }

  // Both the attribute and the variables are written on every apply. The
  // variables are recomputed rather than cached because the tones a seed
  // resolves to depend on the theme type, so switching light to dark with the
  // same colour is a different set of values.
  private apply(): void {
    const theme = this.definition();
    const root = this.document.documentElement;
    root.dataset['theme'] = theme.id;

    if (!theme.customisable) {
      // The document persists across a theme switch in a running app — it is
      // never torn down between selections — so a compiled-palette theme must
      // actively clear any variables a prior customisable selection left set,
      // or they linger as inline styles no stylesheet can override.
      this.clear('primary');
      this.clear('accent');
      return;
    }

    const primary = toneSetFrom(this.colour(), theme.type);
    // The accent is the seed's complement, which keeps a user-picked colour
    // from producing a theme whose tertiary role is indistinguishable from its
    // primary — the failure the monochrome default made unavoidable.
    const accent = toneSetFrom(this.complementOf(this.colour()), theme.type);

    this.publish('primary', primary);
    this.publish('accent', accent);
  }

  private publish(role: 'primary' | 'accent', tones: ToneSet): void {
    const style = this.document.documentElement.style;
    style.setProperty(`--${role}-500`, tones.base);
    style.setProperty(`--${role}-contrast-500`, tones.on);
    style.setProperty(`--${role}-container-500`, tones.container);
    style.setProperty(`--${role}-container-contrast-500`, tones.onContainer);
  }

  private clear(role: 'primary' | 'accent'): void {
    const style = this.document.documentElement.style;
    style.removeProperty(`--${role}-500`);
    style.removeProperty(`--${role}-contrast-500`);
    style.removeProperty(`--${role}-container-500`);
    style.removeProperty(`--${role}-container-contrast-500`);
  }

  private complementOf(hex: string): string {
    const parsed = parseInt(hex.replace('#', ''), 16);
    const rotated = ((parsed >> 8) | ((parsed & 0xff) << 16)) & 0xffffff;
    return `#${rotated.toString(16).padStart(6, '0')}`;
  }

  private restoreTheme(): ThemeId {
    const stored = this.read(THEME_KEY);
    return THEMES.some((theme) => theme.id === stored)
      ? (stored as ThemeId)
      : DEFAULT_THEME_ID;
  }

  private restoreColour(): string {
    const stored = this.read(COLOUR_KEY);
    return stored && /^#[0-9a-f]{6}$/i.test(stored)
      ? stored
      : DEFAULT_CUSTOM_COLOUR;
  }

  // Storage is unavailable in a locked-down browser profile and throws rather
  // than returning null, and a theme preference is not worth failing a boot
  // over.
  private read(key: string): string | null {
    try {
      return this.document.defaultView?.localStorage.getItem(key) ?? null;
    } catch {
      return null;
    }
  }

  private persist(key: string, value: string): void {
    try {
      this.document.defaultView?.localStorage.setItem(key, value);
    } catch {
      return;
    }
  }
}
