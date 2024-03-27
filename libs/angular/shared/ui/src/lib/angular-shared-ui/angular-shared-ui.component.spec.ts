import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AngularSharedUiComponent } from './angular-shared-ui.component';

describe('AngularSharedUiComponent', () => {
  let component: AngularSharedUiComponent;
  let fixture: ComponentFixture<AngularSharedUiComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AngularSharedUiComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(AngularSharedUiComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
