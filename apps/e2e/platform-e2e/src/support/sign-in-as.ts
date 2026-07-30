import { Page } from '@playwright/test';

// Built rather than pasted: a literal JWT is a high-entropy string that reads
// as a real credential to secret scanners, and the claims are the point here.
const segment = (value: object) =>
  btoa(JSON.stringify(value)).replace(/=+$/, '');
const unsignedToken = (claims: object) =>
  `${segment({ alg: 'none', typ: 'JWT' })}.${segment(claims)}.`;

const USER_ID = 42;

export type SignedInUser = {
  /** Omit to exercise the initials avatar; supply base64 PNG data for a photo. */
  iconData?: string;
  firstName?: string;
  lastName?: string;
};

/**
 * Puts the shell in its signed-in state without a backend: seeds the `jwtToken`
 * cookie the AuthService reads and fulfils the single user lookup the account
 * menu derives its view from.
 *
 * Must be called before the navigation that renders the shell — the token is
 * read while the app bootstraps.
 */
export async function signInAs(
  page: Page,
  user: SignedInUser = {},
): Promise<void> {
  const { firstName = 'Jane', lastName = 'Doe', iconData } = user;

  await page.context().addCookies([
    {
      name: 'jwtToken',
      // exp 2033: far enough out that the shell's own expiry check passes
      // without the fixture needing a clock.
      value: unsignedToken({ user_id: USER_ID, exp: 2_000_000_000 }),
      // Resolved the same way the Playwright config resolves baseURL: this runs
      // before the first navigation, so the page has no URL to borrow yet.
      url: process.env['BASE_URL'] ?? 'http://localhost:4200',
    },
  ]);

  await page.route(`**/api/users/${USER_ID}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID,
        emailAddress: 'jane.doe@example.com',
        profile: {
          id: 7,
          firstName,
          lastName,
          icon: iconData ? { id: 3, profileId: 7, icon: iconData } : null,
        },
      }),
    });
  });
}

/** Smallest valid PNG, so the photo avatar has real image bytes to lay out. */
export const ONE_PIXEL_PNG =
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg==';
