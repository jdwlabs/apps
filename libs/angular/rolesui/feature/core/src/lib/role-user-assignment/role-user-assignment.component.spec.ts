import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RoleUserAssignmentComponent } from './role-user-assignment.component';

describe('RoleUserAssignmentComponent', () => {
  let component: RoleUserAssignmentComponent;
  let fixture: ComponentFixture<RoleUserAssignmentComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RoleUserAssignmentComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(RoleUserAssignmentComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
