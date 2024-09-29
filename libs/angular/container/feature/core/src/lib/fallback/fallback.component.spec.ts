import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FallbackComponent } from './fallback.component';
import { ActivatedRoute } from '@angular/router';

describe('FallbackComponent', () => {
  let component: FallbackComponent;
  let fixture: ComponentFixture<FallbackComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FallbackComponent],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(FallbackComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
