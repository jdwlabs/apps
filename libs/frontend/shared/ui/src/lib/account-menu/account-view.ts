import { User } from '@jdw/frontend-shared-util';

export type AvatarState = 'icon' | 'initials' | 'signed-out';

export interface AccountView {
  user: User | null;
  isLoggedIn: boolean;
  avatarState: AvatarState;
  iconData: string;
  displayName: string;
  initials: string;
}

export const SIGNED_OUT_VIEW: AccountView = {
  user: null,
  isLoggedIn: false,
  avatarState: 'signed-out',
  iconData: '',
  displayName: '',
  initials: '',
};

function displayNameOf(user: User): string {
  const profile = user.profile;
  if (!profile) {
    return user.emailAddress ?? '';
  }
  return `${profile.firstName} ${profile.lastName}`.trim();
}

// Without initials an account with no uploaded icon is indistinguishable from
// every other account with no uploaded icon.
function initialsOf(user: User): string {
  const profile = user.profile;
  if (!profile) {
    return (user.emailAddress ?? '?').charAt(0).toUpperCase();
  }
  const first = profile.firstName?.charAt(0) ?? '';
  const last = profile.lastName?.charAt(0) ?? '';
  return `${first}${last}`.toUpperCase() || '?';
}

// Pure, and deliberately free of any service: a `type:ui` lib cannot reach
// `type:data-access` — shared-data-access already depends on shared-ui, so the
// import would close a cycle.
export function accountViewOf(user: User): AccountView {
  return {
    user,
    isLoggedIn: true,
    avatarState: user.profile?.icon?.icon ? 'icon' : 'initials',
    iconData: `data:image/png;base64,${user.profile?.icon?.icon ?? ''}`,
    displayName: displayNameOf(user),
    initials: initialsOf(user),
  };
}
