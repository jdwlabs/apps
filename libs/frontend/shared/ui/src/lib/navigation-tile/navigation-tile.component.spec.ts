import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NavigationTileComponent } from './navigation-tile.component';
import { ActivatedRoute } from '@angular/router';

describe('NavigationTileComponent', () => {
  let component: NavigationTileComponent;
  let fixture: ComponentFixture<NavigationTileComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationTileComponent],
      providers: [{ provide: ActivatedRoute, useValue: ActivatedRoute }],
    }).compileComponents();

    fixture = TestBed.createComponent(NavigationTileComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
