import { Component, inject, ChangeDetectionStrategy } from '@angular/core';

import { NavigationTile } from '@jdw/frontend-shared-util';
import { NavigationTileComponent } from '@jdw/frontend-shared-ui';
import { AuthService } from '@jdw/frontend-shared-data-access';

@Component({
  selector: 'lib-dashboard',
  imports: [NavigationTileComponent],
  templateUrl: './dashboard.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './dashboard.component.scss',
})
export class DashboardComponent {
  private authService = inject(AuthService);

  // A getter, not a field: as a field initialiser the current-user link is
  // built once at construction and renders './user/' whenever the token is not
  // readable yet.
  get navigationTiles(): NavigationTile[] {
    return [
      {
        title: 'Users',
        link: './users',
        description: 'This is a page for viewing all the users',
        icon: 'groups',
      },
      {
        title: 'Profiles',
        link: './profiles',
        description: 'This is a page for viewing all the profiles',
        icon: 'badge',
      },
      {
        title: 'Current User',
        link: './user/' + this.authService.getUserIdFromToken(),
        description: 'This is a page for viewing the current logged in user',
        icon: 'account_circle',
      },
      {
        title: 'Add User',
        link: './account',
        description: 'This is a page for adding a new user account',
        icon: 'person_add',
      },
    ];
  }
}
