import { Component, Input } from '@angular/core';

import { MatButton } from '@angular/material/button';
import { MatIcon } from '@angular/material/icon';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'jdw-fallback',
  imports: [MatButton, MatIcon, RouterLink],
  templateUrl: './fallback.component.html',
  styleUrl: './fallback.component.scss',
})
export class FallbackComponent {
  @Input() redirectLink = '/home';
  @Input() redirectIcon = 'home';
  @Input() redirectText = 'Home';

  onReload() {
    window.location.reload();
  }
}
