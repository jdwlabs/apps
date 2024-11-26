import { Route } from '@angular/router';
import { UsersComponent } from './users/users.component';
import { UserComponent } from './user/user.component';
import { DashboardComponent } from './dashboard/dashboard.component';

export const angularUsersuiFeatureCoreRoutes: Route[] = [
  { path: '', component: DashboardComponent },
  { path: 'user/:userId', component: UserComponent },
  { path: 'users', component: UsersComponent },
  { path: '**', redirectTo: '' },
];
