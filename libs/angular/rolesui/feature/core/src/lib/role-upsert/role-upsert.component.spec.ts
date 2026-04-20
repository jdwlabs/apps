import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RoleUpsertComponent } from './role-upsert.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { ActivatedRoute, convertToParamMap } from '@angular/router';

describe('RoleUpsertComponent', () => {
  let component: RoleUpsertComponent;
  let fixture: ComponentFixture<RoleUpsertComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RoleUpsertComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: ENVIRONMENT,
        },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: convertToParamMap({ roleId: '1' }),
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RoleUpsertComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
