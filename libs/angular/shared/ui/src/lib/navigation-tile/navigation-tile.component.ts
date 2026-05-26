import { Component, Input } from '@angular/core';

import { RouterLink } from '@angular/router';
import { MatCard } from '@angular/material/card';

@Component({
  selector: 'jdw-navigation-tile',
  imports: [RouterLink, MatCard],
  templateUrl: './navigation-tile.component.html',
  styleUrl: './navigation-tile.component.scss',
})
export class NavigationTileComponent {
  @Input() title = '';
  @Input() description = '';
  @Input() link = '/';
}
