import { Route } from '@angular/router';
import { UsersComponent } from './users/users.component';
import { UserComponent } from './user/user.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import { ProfilesComponent } from './profiles/profiles.component';
import { ProfileComponent } from './profile/profile.component';
import { AddressComponent } from './address/address.component';
import { AccountComponent } from './account/account.component';
import { IconComponent } from './icon/icon.component';

export const angularUsersuiFeatureCoreRoutes: Route[] = [
  { path: '', component: DashboardComponent },
  { path: 'user/:userId', component: UserComponent },
  { path: 'users', component: UsersComponent },
  { path: 'profile/:userId', component: ProfileComponent },
  { path: 'profiles', component: ProfilesComponent },
  {
    path: 'profile/:profileId/address/:addressId',
    component: AddressComponent,
  },
  {
    path: 'profile/:profileId/address',
    component: AddressComponent,
  },
  {
    path: 'account/:userId',
    component: AccountComponent,
  },
  {
    path: 'account',
    component: AccountComponent,
  },
  {
    path: 'profile/:profileId/icon',
    component: IconComponent,
  },
  { path: '**', redirectTo: '' },
];
