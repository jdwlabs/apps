import {
  Component,
  EventEmitter,
  Input,
  Output,
  ChangeDetectionStrategy,
} from '@angular/core';
import { MatIconButton } from '@angular/material/button';
import { MatMenu, MatMenuItem, MatMenuTrigger } from '@angular/material/menu';
import { MatIcon } from '@angular/material/icon';
import { MatTooltip } from '@angular/material/tooltip';
import { AccountView, SIGNED_OUT_VIEW } from './account-view';

// Presentational only. The token stream and user lookup live in the feature
// that hosts this, because a `type:ui` lib cannot depend on `type:data-access`.
@Component({
  selector: 'jdw-account-menu',
  imports: [
    MatIconButton,
    MatMenu,
    MatMenuItem,
    MatMenuTrigger,
    MatIcon,
    MatTooltip,
  ],
  templateUrl: './account-menu.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './account-menu.component.scss',
})
export class AccountMenuComponent {
  // A null view is the pre-resolution state and renders as signed out, so the
  // trigger never flickers between shapes while the user request is in flight.
  @Input() set view(value: AccountView | null) {
    this.current = value ?? SIGNED_OUT_VIEW;
  }

  @Output() account = new EventEmitter<void>();
  @Output() profile = new EventEmitter<void>();
  @Output() signOut = new EventEmitter<void>();
  @Output() signIn = new EventEmitter<void>();
  @Output() signUp = new EventEmitter<void>();

  current: AccountView = SIGNED_OUT_VIEW;
}
