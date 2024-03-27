import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NavigationItem } from '@jdw/angular-shared-util';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'jdw-navigation-item',
  standalone: true,
  imports: [
    CommonModule,
    MatIconModule,
    MatListModule,
    RouterLinkActive,
    RouterLink,
  ],
  templateUrl: './navigation-item.component.html',
  styleUrl: './navigation-item.component.scss',
})
export class NavigationItemComponent {
  @Input() navigationItem: NavigationItem = {
    path: '',
    icon: '',
    title: '',
  };
  @Input() isExpanded = false;
  // TODO: Add and figure out how the styles will work with a library _navigation-item-theme.scss
}
