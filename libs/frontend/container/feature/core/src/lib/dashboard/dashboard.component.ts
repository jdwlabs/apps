import {
  Component,
  inject,
  OnInit,
  ChangeDetectionStrategy,
} from '@angular/core';

import { NavigationTileComponent } from '@jdw/frontend-shared-ui';
import { NavigationTile } from '@jdw/frontend-shared-util';
import { MicroFrontendService } from '@jdw/frontend-container-data-access';

@Component({
  selector: 'jdw-dashboard',
  imports: [NavigationTileComponent],
  templateUrl: './dashboard.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './dashboard.component.scss',
})
export class DashboardComponent implements OnInit {
  private microFrontendService: MicroFrontendService =
    inject(MicroFrontendService);

  navigationTiles: NavigationTile[] = [];

  ngOnInit() {
    this.microFrontendService.getRoutes().subscribe((routes) => {
      routes.forEach((route) => {
        this.navigationTiles.push({
          title: route.title,
          description: route.description,
          link: route.path,
        });
      });
    });
  }
}
