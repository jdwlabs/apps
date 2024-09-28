import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MainComponent } from './main.component';
import { ActivatedRoute } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';

describe('MainComponent', () => {
  let component: MainComponent;
  let fixture: ComponentFixture<MainComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MainComponent, NoopAnimationsModule, HttpClientTestingModule],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: {},
        },
        {
          provide: ENVIRONMENT,
          useValue: {
            ENVIRONMENT: 'test',
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(MainComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
