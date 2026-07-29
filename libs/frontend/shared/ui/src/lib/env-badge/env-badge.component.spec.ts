import { ComponentFixture, TestBed } from '@angular/core/testing';
import { EnvBadgeComponent } from './env-badge.component';

describe('EnvBadgeComponent', () => {
  let component: EnvBadgeComponent;
  let fixture: ComponentFixture<EnvBadgeComponent>;

  const text = (selector: string) =>
    (fixture.nativeElement as HTMLElement)
      .querySelector(selector)
      ?.textContent?.trim();

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EnvBadgeComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(EnvBadgeComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('renders the environment uppercased and the version verbatim', () => {
    component.environment = 'non';
    component.version = '2.1.0';
    fixture.detectChanges();

    expect(text('[data-cy="env-segment"]')).toBe('NON');
    expect(text('[data-cy="version-segment"]')).toBe('2.1.0');
  });

  it('maps the deployed production value "prod" to the prod tone', () => {
    component.environment = 'prod';
    fixture.detectChanges();

    expect(component.tone).toBe('prod');
  });

  it('maps the local value to the local tone', () => {
    component.environment = 'LOCAL';
    fixture.detectChanges();

    expect(component.tone).toBe('local');
  });

  it('maps an unknown environment to the non tone rather than prod', () => {
    component.environment = 'staging';
    fixture.detectChanges();

    expect(component.tone).toBe('non');
  });

  it('does not treat "prd" as production, since no chart emits it', () => {
    component.environment = 'prd';
    fixture.detectChanges();

    expect(component.tone).toBe('non');
  });

  it('exposes one accessible name for the pair', () => {
    component.environment = 'non';
    component.version = '2.1.0';
    fixture.detectChanges();

    const badge = (fixture.nativeElement as HTMLElement).querySelector(
      '[data-cy="env-badge"]',
    );

    expect(badge?.getAttribute('aria-label')).toBe(
      'Environment non, version 2.1.0',
    );
  });

  it('applies the tone as a class so the colour follows the environment', () => {
    component.environment = 'prod';
    fixture.detectChanges();

    const badge = (fixture.nativeElement as HTMLElement).querySelector(
      '[data-cy="env-badge"]',
    );

    expect(badge?.classList.contains('prod')).toBe(true);
  });
});
