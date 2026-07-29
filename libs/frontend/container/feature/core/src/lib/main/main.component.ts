import {
  Component,
  inject,
  OnInit,
  ChangeDetectionStrategy,
} from '@angular/core';
import { ThemePalette } from '@angular/material/core';
import { MatDrawerMode } from '@angular/material/sidenav';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { AsyncPipe } from '@angular/common';
import {
  AccountMenuComponent,
  AccountView,
  accountViewOf,
  EnvBadgeComponent,
  HeaderComponent,
  NavigationLayoutComponent,
  SIGNED_OUT_VIEW,
} from '@jdw/frontend-shared-ui';
import { Router, RouterOutlet } from '@angular/router';
import { map, Observable, of, switchMap } from 'rxjs';
import { AuthService, UsersService } from '@jdw/frontend-shared-data-access';
import {
  Environment,
  ENVIRONMENT,
  NavigationItem,
} from '@jdw/frontend-shared-util';
import {
  MicroFrontendService,
  VersionService,
} from '@jdw/frontend-container-data-access';

@Component({
  selector: 'jdw-main',
  imports: [
    AsyncPipe,
    AccountMenuComponent,
    EnvBadgeComponent,
    HeaderComponent,
    NavigationLayoutComponent,
    RouterOutlet,
  ],
  templateUrl: './main.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
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
  private authService: AuthService = inject(AuthService);
  private usersService: UsersService = inject(UsersService);
  private router: Router = inject(Router);

  // A derived stream rather than a subscription writing to a field: token$ is a
  // BehaviorSubject and emits during subscribe, so assigning from that callback
  // would mutate the component in the middle of a change-detection pass.
  readonly accountView$: Observable<AccountView> = this.authService.token$.pipe(
    switchMap((token) => {
      if (!token || this.authService.isTokenExpired(token)) {
        // A stale cookie left in place keeps every guarded request 401ing
        // with nothing telling the user their session ended.
        if (token) {
          this.authService.signOut(false);
        }
        return of(SIGNED_OUT_VIEW);
      }
      const userId = this.authService.getUserIdFromToken(token);
      if (!userId) {
        return of(SIGNED_OUT_VIEW);
      }
      return this.usersService.getUser(userId).pipe(map(accountViewOf));
    }),
  );

  get currentEnv() {
    return this.environment.ENVIRONMENT;
  }

  // Null-tolerant on purpose: the menu only renders these actions when it has a
  // user, but that is the child's invariant, and a template assertion here
  // would turn a change to it into a runtime failure rather than a no-op.
  goToAccount(view: AccountView | null) {
    if (!view?.user) {
      return;
    }
    this.router.navigate([`/users/account/${view.user.id}`]);
  }

  goToProfile(view: AccountView | null) {
    if (!view?.user) {
      return;
    }
    this.router.navigate([`/users/profile/${view.user.id}`]);
  }

  signOut() {
    this.authService.signOut();
    this.router.navigate(['/auth/sign-in']);
  }

  signIn() {
    this.router.navigate(['/auth/sign-in']);
  }

  signUp() {
    this.router.navigate(['/auth/sign-up']);
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
