import { Route } from '@angular/router';
import { UsersComponent } from './users/users.component';
import { UserComponent } from './user/user.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import { ProfilesComponent } from './profiles/profiles.component';

export const angularUsersuiFeatureCoreRoutes: Route[] = [
  { path: '', component: DashboardComponent },
  { path: 'user/:userId', component: UserComponent },
  { path: 'users', component: UsersComponent },
  { path: 'profiles', component: ProfilesComponent },
  { path: '**', redirectTo: '' },
];
