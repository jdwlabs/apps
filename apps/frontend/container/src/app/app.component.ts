import { Component, ChangeDetectionStrategy } from '@angular/core';
import { RouterModule } from '@angular/router';
import { MainComponent } from '@jdw/frontend-container-feature-core';

@Component({
  imports: [RouterModule, MainComponent],
  selector: 'jdw-root',
  templateUrl: './app.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './app.component.scss',
})
export class AppComponent {
  title = 'container';
}
