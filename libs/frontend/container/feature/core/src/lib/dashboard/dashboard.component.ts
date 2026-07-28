import {
  Component,
  inject,
  OnInit,
  ChangeDetectionStrategy,
} from '@angular/core';

import { MatIcon } from '@angular/material/icon';
import { NavigationTileComponent } from '@jdw/frontend-shared-ui';
import { NavigationTile } from '@jdw/frontend-shared-util';
import { MicroFrontendService } from '@jdw/frontend-container-data-access';

@Component({
  selector: 'jdw-dashboard',
  imports: [NavigationTileComponent, MatIcon],
  templateUrl: './dashboard.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './dashboard.component.scss',
})
export class DashboardComponent implements OnInit {
  private microFrontendService: MicroFrontendService =
    inject(MicroFrontendService);

  navigationTiles: NavigationTile[] = [];
  isLoading = true;

  // Placeholder count for the loading state. Three is the steady-state number of
  // micro-frontends; the skeletons are only there to hold the grid's shape.
  readonly skeletons = [0, 1, 2];

  ngOnInit() {
    // getRoutes() reports transport failures through a snackbar and completes
    // with an empty array, so a failed fetch lands in the empty branch rather
    // than needing an error branch of its own.
    this.microFrontendService.getRoutes().subscribe((routes) => {
      this.navigationTiles = routes.map((route) => ({
        title: route.title,
        description: route.description,
        link: route.path,
        icon: route.icon,
      }));
      this.isLoading = false;
    });
  }
}
