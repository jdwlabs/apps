import { ComponentFixture, TestBed } from '@angular/core/testing';
import { UsersActionsButtonCellRendererComponent } from './users-actions-button-cell-renderer.component';
import { ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/frontend-shared-util';
import { MatDialog } from '@angular/material/dialog';
import { HttpTestingController } from '@angular/common/http/testing';
import { of } from 'rxjs';
import { ICellRendererParams } from 'ag-grid-community';

const dialogOpen = vi.fn(() => ({ afterClosed: () => of(undefined) }));

describe('ActionsButtonCellRendererComponent', () => {
  let component: UsersActionsButtonCellRendererComponent;
  let fixture: ComponentFixture<UsersActionsButtonCellRendererComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UsersActionsButtonCellRendererComponent],
      providers: [
        { provide: MatDialog, useValue: { open: dialogOpen } },
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: { AUTH_BASE_URL: 'http://localhost:8080' },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(UsersActionsButtonCellRendererComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('loads the profile for the delete dialog when the row only has profileId', () => {
    const httpTesting = TestBed.inject(HttpTestingController);
    component.agInit({
      data: { id: 5, emailAddress: 'jane@example.com', profileId: 7 },
    } as unknown as ICellRendererParams);

    component.deleteUser();
    httpTesting
      .expectOne('http://localhost:8080/api/profiles/by-user/5')
      .flush({ id: 7, firstName: 'Jane', addresses: [] });

    expect(dialogOpen).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        data: expect.objectContaining({
          profile: expect.objectContaining({ firstName: 'Jane' }),
        }),
      }),
    );
  });
});
