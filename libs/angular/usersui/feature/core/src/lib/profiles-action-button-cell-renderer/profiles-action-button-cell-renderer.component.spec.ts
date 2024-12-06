import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ProfilesActionButtonCellRendererComponent } from './profiles-action-button-cell-renderer.component';

describe('ProfilesActionButtonCellRendererComponent', () => {
  let component: ProfilesActionButtonCellRendererComponent;
  let fixture: ComponentFixture<ProfilesActionButtonCellRendererComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProfilesActionButtonCellRendererComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(
      ProfilesActionButtonCellRendererComponent,
    );
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
