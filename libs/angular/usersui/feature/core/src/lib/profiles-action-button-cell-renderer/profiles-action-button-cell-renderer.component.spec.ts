import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ProfilesActionButtonCellRendererComponent } from './profiles-action-button-cell-renderer.component';
import { ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';

describe('ProfilesActionButtonCellRendererComponent', () => {
  let component: ProfilesActionButtonCellRendererComponent;
  let fixture: ComponentFixture<ProfilesActionButtonCellRendererComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProfilesActionButtonCellRendererComponent],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: {},
        },
      ],
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
