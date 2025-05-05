import { Route } from '@angular/router';
import { DashboardComponent } from './dashboard/dashboard.component';
import { RoleComponent } from './role/role.component';
import { RolesComponent } from './roles/roles.component';
import { RoleUpsertComponent } from './role-upsert/role-upsert.component';

export const angularRolesuiFeatureCoreRoutes: Route[] = [
  { path: '', component: DashboardComponent },
  { path: 'role', component: RoleUpsertComponent },
  { path: 'role/:roleId/edit', component: RoleUpsertComponent },
  { path: 'role/:roleId', component: RoleComponent },
  { path: 'roles', component: RolesComponent },
];
