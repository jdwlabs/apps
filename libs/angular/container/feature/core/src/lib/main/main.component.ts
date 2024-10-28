import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ThemePalette } from '@angular/material/core';
import { MatDrawerMode } from '@angular/material/sidenav';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import {
  HeaderComponent,
  NavigationLayoutComponent,
} from '@jdw/angular-shared-ui';
import { RouterOutlet } from '@angular/router';
import {
  Environment,
  ENVIRONMENT,
  NavigationItem,
} from '@jdw/angular-shared-util';
import {
  MatChip,
  MatChipListbox,
  MatChipOption,
} from '@angular/material/chips';
import { MatButton } from '@angular/material/button';
import {
  MicroFrontendService,
  VersionService,
} from '@jdw/angular-container-data-access';

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
export class MainComponent implements OnInit {
  appTitle = 'JDW';
  appRouterLink = '/';
  appTooltip = 'Home';
  headerColor: ThemePalette = 'primary';
  isXSmallScreen = false;
  isSideNavOpened = false;
  sideNavMode: MatDrawerMode = 'side';
  isSideNavEnabled = true;
  navigationItems: NavigationItem[] = [];
  currentVersion = '';
  private environment: Environment = inject(ENVIRONMENT);
  private breakpointObserver: BreakpointObserver = inject(BreakpointObserver);
  private versionService: VersionService = inject(VersionService);
  private microFrontendService: MicroFrontendService =
    inject(MicroFrontendService);

  get currentEnv() {
    return this.environment.ENVIRONMENT;
  }

  ngOnInit() {
    this.breakpointObserver.observe(Breakpoints.XSmall).subscribe((result) => {
      this.isXSmallScreen = result.matches;
      this.updateNavigationBasedOnScreenSize();
    });
    this.versionService.getVersion().subscribe((version) => {
      this.currentVersion = version;
    });
    this.microFrontendService.getRoutes().subscribe((routes) => {
      const navigationItems: NavigationItem[] = routes.map((route) => ({
        path: route.path,
        icon: route.icon,
        title: route.title,
      }));
      this.navigationItems = [...this.navigationItems, ...navigationItems];
    });
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
