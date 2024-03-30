import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ThemePalette } from '@angular/material/core';
import { MatDrawerMode } from '@angular/material/sidenav';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import {
  HeaderComponent,
  NavigationLayoutComponent,
} from '@jdw/angular-shared-ui';
import { RouterOutlet } from '@angular/router';
import { NavigationItem } from '@jdw/angular-shared-util';
import {
  MatChip,
  MatChipListbox,
  MatChipOption,
} from '@angular/material/chips';
import { MatButton } from '@angular/material/button';
import { ENVIRONMENT, Environment } from '@jdw/angular-container-util';

@Component({
  selector: 'jdw-main',
  standalone: true,
  imports: [
    CommonModule,
    HeaderComponent,
    NavigationLayoutComponent,
    RouterOutlet,
    MatChip,
    MatButton,
    MatChipListbox,
    MatChipOption,
  ],
  templateUrl: './main.component.html',
  styleUrl: './main.component.scss',
})
export class MainComponent {
  appTitle = 'JDW';
  appRouterLink = '/';
  appTooltip = 'Home';
  headerColor: ThemePalette = 'primary';
  isXSmallScreen = false;
  isSideNavOpened = false;
  sideNavMode: MatDrawerMode = 'side';
  isSideNavEnabled = true;
  navigationItems: NavigationItem[] = [
    {
      path: '',
      icon: 'home',
      title: 'Home',
    },
  ];
  currentEnv: string;

  constructor(
    @Inject(ENVIRONMENT) private environment: Environment,
    private breakpointObserver: BreakpointObserver,
  ) {
    this.breakpointObserver.observe(Breakpoints.XSmall).subscribe((result) => {
      this.isXSmallScreen = result.matches;
      this.updateNavigationBasedOnScreenSize();
    });
    this.currentEnv = environment.ENVIRONMENT;
  }

  handleToggle() {
    this.isSideNavOpened = !this.isSideNavOpened;
  }

  sideNavChange(isOpen: boolean) {
    this.isSideNavOpened = isOpen;
  }

  updateNavigationBasedOnScreenSize() {
    if (this.isXSmallScreen) {
      this.sideNavMode = 'over';
      this.isSideNavOpened = false;
    } else {
      this.sideNavMode = 'side';
      this.isSideNavOpened = true;
    }
  }
}
