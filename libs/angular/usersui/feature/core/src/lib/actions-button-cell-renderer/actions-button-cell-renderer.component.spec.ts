import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActionsButtonCellRendererComponent } from './actions-button-cell-renderer.component';

describe('ActionsButtonCellRendererComponent', () => {
  let component: ActionsButtonCellRendererComponent;
  let fixture: ComponentFixture<ActionsButtonCellRendererComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ActionsButtonCellRendererComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(ActionsButtonCellRendererComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
