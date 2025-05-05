import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RolesDialogComponent } from './roles-dialog.component';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';

describe('RolesDialogComponent', () => {
  let component: RolesDialogComponent;
  let fixture: ComponentFixture<RolesDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RolesDialogComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: MAT_DIALOG_DATA,
          useValue: {},
        },
        {
          provide: MatDialogRef,
          useValue: {},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RolesDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
