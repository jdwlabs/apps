import { Component, Input, ChangeDetectionStrategy } from '@angular/core';

type Tone = 'prod' | 'non' | 'local';

@Component({
  selector: 'jdw-env-badge',
  imports: [],
  templateUrl: './env-badge.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './env-badge.component.scss',
})
export class EnvBadgeComponent {
  @Input() environment = '';
  @Input() version = '';

  // Deployed values are LOCAL, non and prod — the production chart spells it
  // "prod", not "prd". Anything unrecognised falls to the non tone: wrongly
  // claiming production is the dangerous direction of the two.
  get tone(): Tone {
    const value = this.environment.toLowerCase();
    if (value === 'prod') {
      return 'prod';
    }
    if (value === 'local') {
      return 'local';
    }
    return 'non';
  }

  get label(): string {
    return `Environment ${this.environment}, version ${this.version}`;
  }
}
