import { Component, ChangeDetectionStrategy } from '@angular/core';

import { RouterOutlet } from '@angular/router';

@Component({
  imports: [RouterOutlet],
  selector: 'app-rolesui-entry',
  changeDetection: ChangeDetectionStrategy.Eager,
  template: ` <router-outlet></router-outlet>`,
})
export class RemoteEntryComponent {}
