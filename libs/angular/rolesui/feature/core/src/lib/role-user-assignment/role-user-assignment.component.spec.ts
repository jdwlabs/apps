import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RoleUserAssignmentComponent } from './role-user-assignment.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';

describe('RoleUserAssignmentComponent', () => {
  let component: RoleUserAssignmentComponent;
  let fixture: ComponentFixture<RoleUserAssignmentComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RoleUserAssignmentComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RoleUserAssignmentComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
