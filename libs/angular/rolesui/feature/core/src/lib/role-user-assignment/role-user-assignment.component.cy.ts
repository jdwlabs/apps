import { TestBed } from '@angular/core/testing';
import { RoleUserAssignmentComponent } from './role-user-assignment.component';

describe(RoleUserAssignmentComponent.name, () => {
  beforeEach(() => {
    TestBed.overrideComponent(RoleUserAssignmentComponent, {
      add: {
        imports: [],
        providers: [],
      },
    });
  });

  it('renders', () => {
    cy.mount(RoleUserAssignmentComponent);
  });
});
