import { Route } from '@angular/router';
import { SignInComponent } from './sign-in/sign-in.component';
import { SignUpComponent } from './sign-up/sign-up.component';
import { ForbiddenComponent } from './forbidden/forbidden.component';

export const angularAuthuiFeatureCoreRoutes: Route[] = [
  { path: 'sign-in', component: SignInComponent },
  { path: 'sign-up', component: SignUpComponent },
  { path: 'forbidden', component: ForbiddenComponent },
  { path: '**', redirectTo: 'sign-in' },
];
