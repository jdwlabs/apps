import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { User } from '@jdw/frontend-shared-util';
import { AccountMenuComponent } from './account-menu.component';
import { accountViewOf, SIGNED_OUT_VIEW } from './account-view';

const userWithoutIcon = {
  id: 42,
  emailAddress: 'jake@example.com',
  profile: { firstName: 'Jake', lastName: 'Willmsen', icon: null },
} as unknown as User;

const userWithIcon = {
  id: 42,
  emailAddress: 'jake@example.com',
  profile: { firstName: 'Jake', lastName: 'Willmsen', icon: { icon: 'QUJD' } },
} as unknown as User;

const userWithoutProfile = {
  id: 42,
  emailAddress: 'jake@example.com',
  profile: null,
} as unknown as User;

describe('accountViewOf', () => {
  it('shows initials when the user has no stored icon', () => {
    const view = accountViewOf(userWithoutIcon);

    expect(view.avatarState).toBe('initials');
    expect(view.initials).toBe('JW');
    expect(view.displayName).toBe('Jake Willmsen');
  });

  it('shows the stored icon when one exists', () => {
    const view = accountViewOf(userWithIcon);

    expect(view.avatarState).toBe('icon');
    expect(view.iconData).toBe('data:image/png;base64,QUJD');
  });

  it('falls back to the email initial when there is no profile', () => {
    const view = accountViewOf(userWithoutProfile);

    expect(view.initials).toBe('J');
    expect(view.displayName).toBe('jake@example.com');
  });
});

describe('AccountMenuComponent', () => {
  let component: AccountMenuComponent;
  let fixture: ComponentFixture<AccountMenuComponent>;

  const has = (selector: string) =>
    !!(fixture.nativeElement as HTMLElement).querySelector(selector);

  const textOf = (selector: string) =>
    (fixture.nativeElement as HTMLElement)
      .querySelector(selector)
      ?.textContent?.trim();

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AccountMenuComponent],
      providers: [provideNoopAnimations()],
    }).compileComponents();

    fixture = TestBed.createComponent(AccountMenuComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('renders the signed-out glyph when no view has arrived yet', () => {
    component.view = null;
    fixture.detectChanges();

    expect(component.current).toEqual(SIGNED_OUT_VIEW);
    expect(has('[data-cy="avatar-signed-out"]')).toBe(true);
  });

  it('renders initials for a user with no stored icon', () => {
    component.view = accountViewOf(userWithoutIcon);
    fixture.detectChanges();

    expect(textOf('[data-cy="avatar-initials"]')).toBe('JW');
  });

  it('renders the stored icon when the user has one', () => {
    component.view = accountViewOf(userWithIcon);
    fixture.detectChanges();

    expect(has('[data-cy="avatar-image"]')).toBe(true);
  });

  it('names the trigger after the account for screen readers', () => {
    component.view = accountViewOf(userWithoutIcon);
    fixture.detectChanges();

    const trigger = (fixture.nativeElement as HTMLElement).querySelector(
      '[data-cy="account-menu-button"]',
    );

    expect(trigger?.getAttribute('aria-label')).toBe(
      'Account menu for Jake Willmsen',
    );
  });

  it('emits sign out rather than acting on it itself', () => {
    component.view = accountViewOf(userWithoutIcon);
    fixture.detectChanges();

    const emitted = vi.fn();
    component.signOut.subscribe(emitted);
    component.signOut.emit();

    expect(emitted).toHaveBeenCalled();
  });
});
