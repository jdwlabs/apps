import { Component } from '@angular/core';
import { RouterModule } from '@angular/router';
import { MainComponent } from '@jdw/angular-container-feature-core';

@Component({
  standalone: true,
  imports: [RouterModule, MainComponent],
  selector: 'jdw-root',
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss',
})
export class AppComponent {
  title = 'container';
}
