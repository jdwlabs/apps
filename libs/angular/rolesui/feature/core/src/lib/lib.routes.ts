import { Route } from '@angular/router';
import { DashboardComponent } from './dashboard/dashboard.component';
import { RoleComponent } from './role/role.component';
import { RolesComponent } from './roles/roles.component';
import { RoleUpsertComponent } from './role-upsert/role-upsert.component';
import { UserRoleAssignmentComponent } from './user-role-assignment/user-role-assignment.component';
import { RoleUserAssignmentComponent } from './role-user-assignment/role-user-assignment.component';

export const angularRolesuiFeatureCoreRoutes: Route[] = [
  { path: '', component: DashboardComponent },
  { path: 'role', component: RoleUpsertComponent },
  { path: 'role/:roleId/edit', component: RoleUpsertComponent },
  { path: 'role/:roleId', component: RoleComponent },
  { path: 'roles', component: RolesComponent },
  { path: 'assign/roles', component: UserRoleAssignmentComponent },
  { path: 'assign/users', component: RoleUserAssignmentComponent },
];
