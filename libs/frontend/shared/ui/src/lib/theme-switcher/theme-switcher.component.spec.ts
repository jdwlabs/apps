import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialog } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ThemeDialogComponent } from './theme-dialog.component';
import { ThemeSwitcherComponent } from './theme-switcher.component';

describe('ThemeSwitcherComponent', () => {
  let fixture: ComponentFixture<ThemeSwitcherComponent>;
  let dialog: MatDialog;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ThemeSwitcherComponent, NoopAnimationsModule],
    }).compileComponents();
    fixture = TestBed.createComponent(ThemeSwitcherComponent);
    dialog = TestBed.inject(MatDialog);
    fixture.detectChanges();
  });

  it('opens the theme dialog when the trigger is clicked', () => {
    const openSpy = vi.spyOn(dialog, 'open');
    fixture.nativeElement.querySelector('[data-cy="theme-trigger"]').click();
    expect(openSpy).toHaveBeenCalledWith(ThemeDialogComponent);
  });
});
