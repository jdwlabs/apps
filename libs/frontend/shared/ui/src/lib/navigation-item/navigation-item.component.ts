import { Component, Input, ChangeDetectionStrategy } from '@angular/core';

import { NavigationItem } from '@jdw/frontend-shared-util';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';

@Component({
  selector: 'jdw-navigation-item',
  imports: [
    MatIconModule,
    MatListModule,
    MatTooltipModule,
    RouterLinkActive,
    RouterLink,
  ],
  templateUrl: './navigation-item.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './navigation-item.component.scss',
})
export class NavigationItemComponent {
  @Input() navigationItem: NavigationItem = {
    path: '',
    icon: '',
    title: '',
  };
  @Input() isExpanded = false;
}
