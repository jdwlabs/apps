import { Route } from '@angular/router';
import { UsersComponent } from './users/users.component';

export const angularUsersuiFeatureCoreRoutes: Route[] = [
  { path: 'users', component: UsersComponent },
  { path: '**', redirectTo: 'users' },
];
