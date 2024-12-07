import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatListModule } from '@angular/material/list';
import { MatDrawerMode, MatSidenavModule } from '@angular/material/sidenav';
import { NavigationItemComponent } from '../navigation-item/navigation-item.component';
import { NavigationItem } from '@jdw/angular-shared-util';

@Component({
  selector: 'jdw-navigation-layout',
  imports: [
    CommonModule,
    MatSidenavModule,
    MatListModule,
    MatTooltipModule,
    MatButtonModule,
    MatIconModule,
    NavigationItemComponent,
  ],
  templateUrl: './navigation-layout.component.html',
  styleUrl: './navigation-layout.component.scss',
})
export class NavigationLayoutComponent {
  @Input() isSideNavEnabled = true;
  @Input() sideNavMode: MatDrawerMode = 'side';
  @Input() isSideNavOpened = true;
  @Input() navigationItems: NavigationItem[] = [];
  @Output() sideNavChange = new EventEmitter<boolean>();
  isExpanded = false;

  toggleSideNav() {
    this.isExpanded = !this.isExpanded;
  }

  openedChanged(isOpen: boolean) {
    this.sideNavChange.emit(isOpen);
  }
}
