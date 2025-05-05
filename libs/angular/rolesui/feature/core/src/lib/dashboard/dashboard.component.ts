import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NavigationTileComponent } from '@jdw/angular-shared-ui';
import { NavigationTile } from '@jdw/angular-shared-util';

@Component({
  selector: 'lib-dashboard',
  imports: [CommonModule, NavigationTileComponent],
  templateUrl: './dashboard.component.html',
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
  ];
}
