import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NavigationTileComponent } from '@jdw/angular-shared-ui';
import { NavigationTile } from '@jdw/angular-shared-util';
import { MicroFrontendService } from '@jdw/angular-container-data-access';

@Component({
  selector: 'jdw-dashboard',
  imports: [CommonModule, NavigationTileComponent],
  templateUrl: './dashboard.component.html',
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
