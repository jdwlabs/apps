import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NavigationLayoutComponent } from './navigation-layout.component';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('NavigationLayoutComponent', () => {
  let component: NavigationLayoutComponent;
  let fixture: ComponentFixture<NavigationLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationLayoutComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(NavigationLayoutComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
