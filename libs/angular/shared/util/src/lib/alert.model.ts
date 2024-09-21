import { PaletteColors } from './palette.model';

export type AlertVariants = 'default' | 'filled' | 'outlined';
export type Variant = {
  display: string;
  value: AlertVariants;
};
export type Icon = {
  display: string;
  value: string;
};
export type Alert = {
  id?: string;
  type?: PaletteColors;
  variant?: AlertVariants;
  message?: string;
  autoClose?: boolean;
  autoCloseTimeout?: number;
  keepAfterRouteChange?: boolean;
  fade?: boolean;
  fadeTime?: number;
  icon?: string;
  closeButton?: boolean;
  maxSize?: number;
};
export type AlertOptions = {
  id?: string;
  autoClose?: boolean;
  autoCloseTimeout?: number;
  keepAfterRouteChange?: boolean;
  icon?: string;
  closeButton?: boolean;
  maxSize?: number;
};
