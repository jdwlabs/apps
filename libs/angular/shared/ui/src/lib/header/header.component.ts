import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { RouterLink } from '@angular/router';
import { ThemePalette } from '@angular/material/core';

@Component({
  selector: 'jdw-header',
  standalone: true,
  imports: [
    CommonModule,
    MatToolbarModule,
    MatIconModule,
    MatButtonModule,
    MatTooltipModule,
    RouterLink,
  ],
  templateUrl: './header.component.html',
  styleUrl: './header.component.scss',
})
export class HeaderComponent {
  @Input() isXSmallScreen = false;
  @Input() appTitle = '';
  @Input() appRouterLink = '';
  @Input() appTooltip = '';
  @Input() color: ThemePalette = 'primary';
  @Output() toggleSideNav = new EventEmitter<boolean>();

  toggle() {
    this.toggleSideNav.emit();
  }
}
