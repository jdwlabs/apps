import { Route } from '@angular/router';
import { UsersComponent } from './users/users.component';
import { UserComponent } from './user/user.component';

export const angularUsersuiFeatureCoreRoutes: Route[] = [
  { path: 'user/:userId', component: UserComponent },
  { path: 'users', component: UsersComponent },
  { path: '**', redirectTo: 'users' },
];
