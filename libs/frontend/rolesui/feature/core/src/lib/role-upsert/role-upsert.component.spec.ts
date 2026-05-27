import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RoleUpsertComponent } from './role-upsert.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/frontend-shared-util';
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

  describe('form validation', () => {
    it('requires name', () => {
      component.form.get('name')!.setValue('');
      expect(component.form.get('name')!.hasError('required')).toBe(true);
    });

    it('requires description', () => {
      component.form.get('description')!.setValue('');
      expect(component.form.get('description')!.hasError('required')).toBe(
        true,
      );
    });

    it('is invalid when empty', () => {
      expect(component.form.valid).toBe(false);
    });

    it('is valid when all fields are filled', () => {
      component.form.get('name')!.setValue('ADMIN');
      component.form.get('description')!.setValue('Admin role');
      expect(component.form.valid).toBe(true);
    });
  });
});
