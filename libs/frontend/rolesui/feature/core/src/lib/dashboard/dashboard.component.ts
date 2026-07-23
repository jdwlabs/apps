import { Component, ChangeDetectionStrategy } from '@angular/core';

import { NavigationTileComponent } from '@jdw/frontend-shared-ui';
import { NavigationTile } from '@jdw/frontend-shared-util';

@Component({
  selector: 'lib-dashboard',
  imports: [NavigationTileComponent],
  templateUrl: './dashboard.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './dashboard.component.scss',
})
export class DashboardComponent {
  navigationTiles: NavigationTile[] = [
    {
      title: 'Roles',
      link: './roles',
      description: 'This is a page for viewing all the roles',
    },
    {
      title: 'Add Role',
      link: './role',
      description: 'This is a page for adding a new role',
    },
    {
      title: 'Assign Roles',
      link: './assign/roles',
      description: 'This is a page for assigning roles to a user',
    },
    {
      title: 'Assign Users',
      link: './assign/users',
      description: 'This is a page for assigning users to a role',
    },
  ];
}
