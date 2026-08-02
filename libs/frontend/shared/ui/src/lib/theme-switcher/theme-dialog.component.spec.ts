import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { ThemeService } from '@jdw/frontend-shared-util';
import { beforeEach, describe, expect, it } from 'vitest';
import { ThemeDialogComponent } from './theme-dialog.component';

describe('ThemeDialogComponent', () => {
  let fixture: ComponentFixture<ThemeDialogComponent>;
  let service: ThemeService;

  beforeEach(async () => {
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [ThemeDialogComponent, NoopAnimationsModule],
    }).compileComponents();
    fixture = TestBed.createComponent(ThemeDialogComponent);
    service = TestBed.inject(ThemeService);
    fixture.detectChanges();
  });

  it('offers every registered theme', () => {
    const options = fixture.nativeElement.querySelectorAll(
      '[data-cy="theme-option"]',
    );
    expect(options.length).toBe(service.themes.length);
  });

  it('marks the currently selected theme as the checked radio', () => {
    fixture.componentInstance.choose('user-custom-dark');
    fixture.detectChanges();

    const options: NodeListOf<HTMLElement> =
      fixture.nativeElement.querySelectorAll('[data-cy="theme-option"]');
    const checked = Array.from(options).filter(
      (option) => option.querySelector('input[type="radio"]')?.['checked'],
    );

    expect(checked).toHaveLength(1);
    expect(checked[0].textContent).toContain('Custom dark');
  });

  it('selects a theme through the service', () => {
    fixture.componentInstance.choose('user-custom-dark');
    expect(service.themeId()).toBe('user-custom-dark');
  });

  it('shows the colour input only for a customisable theme', () => {
    fixture.componentInstance.choose('blue-slate');
    fixture.detectChanges();
    expect(fixture.componentInstance.customisable()).toBe(false);

    fixture.componentInstance.choose('user-custom-light');
    fixture.detectChanges();
    expect(fixture.componentInstance.customisable()).toBe(true);
  });

  it('passes a chosen colour to the service', () => {
    fixture.componentInstance.choose('user-custom-light');
    fixture.componentInstance.recolour('#2f5fa8');
    expect(service.customColour()).toBe('#2f5fa8');
  });
});
