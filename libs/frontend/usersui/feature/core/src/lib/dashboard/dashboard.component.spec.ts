import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DashboardComponent } from './dashboard.component';
import { CookieService } from 'ngx-cookie-service';
import { ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { AuthService } from '@jdw/frontend-shared-data-access';
import { ENVIRONMENT } from '@jdw/frontend-shared-util';

class MockCookieService {
  get(key: string): string {
    if (key === 'jwtToken') {
      return 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzNDU2Iiwicm9sZXMiOlsiYWRtaW4iXX0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c';
    }
    return '';
  }
  set() {
    return;
  }
  delete() {
    return;
  }
}

const environmentMock = {
  ENVIRONMENT: 'test',
  AUTH_BASE_URL: 'http://localhost:8080',
  SERVICE_DISCOVERY_BASE_URL: 'http://localhost:9000',
};

describe('DashboardComponent', () => {
  let component: DashboardComponent;
  let fixture: ComponentFixture<DashboardComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DashboardComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: CookieService, useClass: MockCookieService },
        { provide: ENVIRONMENT, useValue: environmentMock },
        {
          provide: ActivatedRoute,
          useValue: ActivatedRoute,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(DashboardComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('builds the current-user link from the auth service', () => {
    const authService = TestBed.inject(AuthService);
    vi.spyOn(authService, 'getUserIdFromToken').mockReturnValue(7);

    const tile = component.navigationTiles.find(
      (navigationTile) => navigationTile.title === 'Current User',
    );

    expect(tile?.link).toBe('./user/7');
  });

  it('reads the user id from the token rather than the cookie directly', () => {
    const tile = component.navigationTiles.find(
      (navigationTile) => navigationTile.title === 'Current User',
    );

    expect(tile?.link).toBe('./user/123456');
  });
});
