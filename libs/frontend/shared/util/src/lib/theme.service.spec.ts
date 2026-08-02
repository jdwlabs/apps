import { TestBed } from '@angular/core/testing';
import { DOCUMENT } from '@angular/common';
import { beforeEach, describe, expect, it } from 'vitest';
import { ThemeService } from './theme.service';
import { DEFAULT_CUSTOM_COLOUR, DEFAULT_THEME_ID } from './theme.model';
import { complementOf, contrastRatio, toneSetFrom } from './on-color.util';

describe('ThemeService', () => {
  let service: ThemeService;
  let root: HTMLElement;

  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({});
    service = TestBed.inject(ThemeService);
    root = TestBed.inject(DOCUMENT).documentElement;
  });

  it('starts on the default theme', () => {
    expect(service.themeId()).toBe(DEFAULT_THEME_ID);
    expect(root.dataset['theme']).toBe(DEFAULT_THEME_ID);
  });

  it('writes the selected theme to the root element', () => {
    service.select('user-custom-dark');
    expect(root.dataset['theme']).toBe('user-custom-dark');
  });

  it('restores the persisted choice', () => {
    service.select('user-custom-light');
    const restored = TestBed.inject(ThemeService);
    expect(restored.themeId()).toBe('user-custom-light');
  });

  it('ignores a persisted value that is not a known theme', () => {
    localStorage.setItem('jdw.theme', 'not-a-theme');
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({});
    expect(TestBed.inject(ThemeService).themeId()).toBe(DEFAULT_THEME_ID);
  });

  // Every variable assertion selects a customisable theme first. A
  // non-customisable theme takes its colours from a compiled palette and
  // publishes nothing, so asserting against the default would compare two
  // empty strings and pass while proving nothing.
  it('publishes container variables distinct from their base', () => {
    service.select('user-custom-light');
    service.setCustomColour('#7dcf2a');
    const style = root.style;
    expect(style.getPropertyValue('--primary-500')).not.toBe('');
    expect(style.getPropertyValue('--primary-container-500')).not.toBe(
      style.getPropertyValue('--primary-500'),
    );
  });

  it('publishes foregrounds that clear the text floor', () => {
    service.select('user-custom-light');
    service.setCustomColour('#7dcf2a');
    const read = (name: string) => root.style.getPropertyValue(name).trim();
    expect(read('--primary-500')).not.toBe('');
    expect(
      contrastRatio(read('--primary-contrast-500'), read('--primary-500')),
    ).toBeGreaterThanOrEqual(4.5);
    expect(
      contrastRatio(
        read('--primary-container-contrast-500'),
        read('--primary-container-500'),
      ),
    ).toBeGreaterThanOrEqual(4.5);
  });

  // The accent is derived rather than published from a second user input, so
  // nothing on screen distinguishes "no accent was written" from "the accent
  // happens to resemble the primary" — the tertiary roles would just fall back
  // to the compiled palette and still look deliberate.
  it('publishes an accent derived from the seed', () => {
    service.select('user-custom-light');
    service.setCustomColour('#2f5fa8');
    const read = (name: string) => root.style.getPropertyValue(name).trim();
    expect(read('--accent-500')).toBe(
      toneSetFrom(complementOf('#2f5fa8'), 'light').base,
    );
    expect(read('--accent-500')).not.toBe(read('--primary-500'));
  });

  it('publishes nothing for a theme that is not customisable', () => {
    service.select('blue-slate');
    expect(root.style.getPropertyValue('--primary-500')).toBe('');
  });

  it('recomputes the variables when the theme type flips', () => {
    service.select('user-custom-light');
    service.setCustomColour('#7dcf2a');
    const light = root.style.getPropertyValue('--primary-500');
    expect(light).not.toBe('');

    service.select('user-custom-dark');
    expect(root.style.getPropertyValue('--primary-500')).not.toBe(light);
  });

  it('starts from the legacy fallback colour', () => {
    expect(service.customColour()).toBe(DEFAULT_CUSTOM_COLOUR);
  });
});
