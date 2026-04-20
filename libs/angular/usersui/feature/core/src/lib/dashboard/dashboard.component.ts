import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NavigationTile } from '@jdw/angular-shared-util';
import { NavigationTileComponent } from '@jdw/angular-shared-ui';
import { CookieService } from 'ngx-cookie-service';
import { jwtDecode, JwtPayload } from 'jwt-decode';

@Component({
  selector: 'lib-dashboard',
  imports: [CommonModule, NavigationTileComponent],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.scss',
})
export class DashboardComponent {
  private cookieService = inject(CookieService);

  navigationTiles: NavigationTile[] = [
    {
      title: 'Users',
      link: './users',
      description: 'This is a page for viewing all the users',
    },
    {
      title: 'Profiles',
      link: './profiles',
      description: 'This is a page for viewing all the profiles',
    },
    {
      title: 'Current User',
      link: './user/' + this.getCurrentUserId(),
      description: 'This is a page for viewing the current logged in user',
    },
    {
      title: 'Add User',
      link: './account',
      description: 'This is a page for adding a new user account',
    },
  ];

  getCurrentUser(): JwtPayload & { user_id?: string; roles?: string[] } {
    const token = this.cookieService.get('jwtToken');
    if (!token) {
      return {};
    }
    return jwtDecode(token);
  }

  getCurrentUserId(): string {
    return this.getCurrentUser().user_id ?? '';
  }
}
