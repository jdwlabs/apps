import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { MatCard } from '@angular/material/card';

@Component({
  selector: 'jdw-navigation-tile',
  imports: [CommonModule, RouterLink, MatCard],
  templateUrl: './navigation-tile.component.html',
  styleUrl: './navigation-tile.component.scss',
})
export class NavigationTileComponent {
  @Input() title = '';
  @Input() description = '';
  @Input() link = '/';
}
