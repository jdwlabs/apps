import { ComponentFixture, TestBed } from '@angular/core/testing';
import { IconComponent } from './icon.component';
import { ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ENVIRONMENT } from '@jdw/angular-shared-util';
import { of } from 'rxjs';
import { ProfilesService } from '@jdw/angular-usersui-data-access';

describe('IconComponent', () => {
  let component: IconComponent;
  let fixture: ComponentFixture<IconComponent>;
  let mockProfilesService: any;

  beforeEach(async () => {
    mockProfilesService = {
      getIcon: jest.fn().mockReturnValue(of(null)),
      addIcon: jest.fn().mockReturnValue(of({ userId: 1 })),
      editIcon: jest.fn().mockReturnValue(of({ userId: 1 })),
    };

    await TestBed.configureTestingModule({
      imports: [IconComponent],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              params: {
                profileId: 1,
              },
            },
          },
        },
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ENVIRONMENT,
          useValue: ENVIRONMENT,
        },
        { provide: ProfilesService, useValue: mockProfilesService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(IconComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with correct type', () => {
    expect(component.type).toBe('Add');
  });

  it('should fetch icon and update type', () => {
    mockProfilesService.getIcon.mockReturnValue(of({}));
    component.ngOnInit();
    expect(component.type).toBe('Edit');
  });

  it('should reject non-image files', () => {
    const file = new File(['dummy'], 'dummy.txt', { type: 'text/plain' });
    component.processFile(file);
    expect(component.errorMessage).toBe(
      'Invalid file type. Please upload an image.',
    );
  });

  it('should reject oversized files', () => {
    const file = new File(['a'.repeat(3 * 1024 * 1024)], 'big.jpg', {
      type: 'image/jpeg',
    });
    Object.defineProperty(file, 'size', { value: 3 * 1024 * 1024 });
    component.processFile(file);
    expect(component.errorMessage).toBe('File size exceeds 2MB limit.');
  });

  it('should set selected file on valid upload', () => {
    const file = new File(['dummy'], 'image.png', { type: 'image/png' });
    const objectURL = 'blob://dummy-url';
    global.URL.createObjectURL = jest.fn().mockReturnValue(objectURL);

    component.processFile(file);

    expect(component.selectedFile).toBeTruthy();
    expect(component.selectedFile?.name).toBe('image.png');
    expect(component.selectedFile?.objectURL).toBe(objectURL);
  });

  it('should call addIcon for new icon submission', () => {
    component.profileId = 1;
    component.selectedFile = Object.assign(
      new File(['dummy'], 'image.png', { type: 'image/png' }),
      {
        objectURL: 'blob://dummy-url',
      },
    );
    component.type = 'Add';
    component.onSubmit();
    expect(mockProfilesService.addIcon).toHaveBeenCalledWith(
      1,
      component.selectedFile,
    );
  });

  it('should call editIcon for existing icon submission', () => {
    component.profileId = 1;
    component.selectedFile = Object.assign(
      new File(['dummy'], 'image.png', { type: 'image/png' }),
      {
        objectURL: 'blob://dummy-url',
      },
    );
    component.type = 'Edit';
    component.onSubmit();
    expect(mockProfilesService.editIcon).toHaveBeenCalledWith(
      1,
      component.selectedFile,
    );
  });

  it('should reset the file and error message on reset', () => {
    component.selectedFile = Object.assign(
      new File(['dummy'], 'image.png', { type: 'image/png' }),
      {
        objectURL: 'blob://dummy-url',
      },
    );
    component.errorMessage = 'Some error';
    component.onReset();
    expect(component.selectedFile).toBeNull();
    expect(component.errorMessage).toBe('');
  });
});
