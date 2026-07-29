import { themeQuartz } from 'ag-grid-community';

/**
 * Grid theme wired to the Material 3 system tokens.
 *
 * Every value is a `var(--mat-sys-*)` reference rather than a literal, so the
 * grid follows whatever theme is applied to the document. Dark mode needs no
 * separate skin: the tokens themselves flip, which is why this replaced a
 * compile-time light/dark Sass branch.
 *
 * `browserColorScheme: inherit` hands the same decision to native widgets
 * (scrollbars, date pickers, text inputs) — without it they stay light on a
 * dark grid.
 *
 * Parameters are set explicitly rather than left to be derived. The grid
 * normally computes values like the border colour by mixing background and
 * foreground, and it cannot mix through an unresolved `var()` — an omitted
 * parameter falls back to the Quartz literal and reads as an off-palette
 * patch rather than failing visibly.
 *
 * Header and panel surfaces come from `chromeBackgroundColor`: this version
 * has no separate header background parameter, so setting only a row
 * background would leave the header on the Quartz default.
 */
export const jdwGridTheme = themeQuartz.withParams({
  browserColorScheme: 'inherit',

  backgroundColor: 'var(--mat-sys-surface)',
  foregroundColor: 'var(--mat-sys-on-surface)',
  accentColor: 'var(--mat-sys-primary)',
  borderColor: 'var(--mat-sys-outline-variant)',

  chromeBackgroundColor: 'var(--mat-sys-surface-container)',
  headerTextColor: 'var(--mat-sys-on-surface-variant)',
  headerColumnResizeHandleColor: 'var(--mat-sys-outline)',
  headerCellHoverBackgroundColor: 'var(--mat-sys-surface-container-high)',

  oddRowBackgroundColor: 'var(--mat-sys-surface-container-lowest)',
  rowHoverColor: 'var(--mat-sys-surface-container-high)',
  selectedRowBackgroundColor: 'var(--mat-sys-secondary-container)',

  menuBackgroundColor: 'var(--mat-sys-surface-container)',
  inputBackgroundColor: 'var(--mat-sys-surface-container-highest)',
  inputTextColor: 'var(--mat-sys-on-surface)',

  fontFamily: 'var(--mat-sys-body-medium-font)',
});
