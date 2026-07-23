import { Component, Input, ChangeDetectionStrategy } from '@angular/core';

import { MatIconModule } from '@angular/material/icon';
import { RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'lib-forbidden',
  imports: [MatIconModule, RouterLink, MatButtonModule],
  templateUrl: './forbidden.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './forbidden.component.scss',
})
export class ForbiddenComponent {
  @Input() redirectLink = '/home';
  @Input() redirectIcon = 'home';
  @Input() redirectText = 'Home';
}
