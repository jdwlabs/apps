import { ThemeType } from './on-color.util';

export type ThemeId = 'blue-slate' | 'user-custom-light' | 'user-custom-dark';

export interface ThemeDefinition {
  readonly id: ThemeId;
  readonly label: string;
  readonly type: ThemeType;
  // Whether the theme takes its key colours from the user rather than from a
  // compiled palette. Only a customisable theme reads the runtime variables.
  readonly customisable: boolean;
}

export const THEMES: readonly ThemeDefinition[] = [
  { id: 'blue-slate', label: 'Blue slate', type: 'light', customisable: false },
  {
    id: 'user-custom-light',
    label: 'Custom light',
    type: 'light',
    customisable: true,
  },
  {
    id: 'user-custom-dark',
    label: 'Custom dark',
    type: 'dark',
    customisable: true,
  },
];

export const DEFAULT_THEME_ID: ThemeId = 'blue-slate';

// The colour the user-custom themes fell back to before a user could choose
// one. Keeping it as the initial value means an existing screen looks the same
// the first time the picker appears.
export const DEFAULT_CUSTOM_COLOUR = '#7dcf2a';
