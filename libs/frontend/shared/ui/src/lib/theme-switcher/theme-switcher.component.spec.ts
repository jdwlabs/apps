import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ThemeService } from '@jdw/frontend-shared-util';
import { beforeEach, describe, expect, it } from 'vitest';
import { ThemeSwitcherComponent } from './theme-switcher.component';

describe('ThemeSwitcherComponent', () => {
  let fixture: ComponentFixture<ThemeSwitcherComponent>;
  let service: ThemeService;

  beforeEach(async () => {
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [ThemeSwitcherComponent],
    }).compileComponents();
    fixture = TestBed.createComponent(ThemeSwitcherComponent);
    service = TestBed.inject(ThemeService);
    fixture.detectChanges();
  });

  it('offers every registered theme', () => {
    fixture.nativeElement.querySelector('[data-cy="theme-trigger"]').click();
    fixture.detectChanges();
    const options = document.querySelectorAll('[data-cy="theme-option"]');
    expect(options.length).toBe(service.themes.length);
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
