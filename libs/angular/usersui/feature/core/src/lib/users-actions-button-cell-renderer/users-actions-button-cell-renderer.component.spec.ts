import { ComponentFixture, TestBed } from '@angular/core/testing';
import { UsersActionsButtonCellRendererComponent } from './users-actions-button-cell-renderer.component';
import { ActivatedRoute } from '@angular/router';

describe('ActionsButtonCellRendererComponent', () => {
  let component: UsersActionsButtonCellRendererComponent;
  let fixture: ComponentFixture<UsersActionsButtonCellRendererComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UsersActionsButtonCellRendererComponent],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(UsersActionsButtonCellRendererComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
