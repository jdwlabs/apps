import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButton, MatIconButton } from '@angular/material/button';
import { MatIcon } from '@angular/material/icon';
import { ActivatedRoute, Router } from '@angular/router';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import { Icon } from '@jdw/angular-usersui-util';

@Component({
  selector: 'lib-icon',
  imports: [CommonModule, MatButton, MatIcon, MatIconButton],
  templateUrl: './icon.component.html',
  styleUrl: './icon.component.scss',
})
export class IconComponent implements OnInit {
  type = 'Add';
  isDragOver = false;
  selectedFile: (File & { objectURL: string }) | null = null;
  errorMessage = '';
  maxFileSize = 2 * 1024 * 1024; // 2 MB limit
  profileId: number | null = null;
  private router: Router = inject(Router);
  private route: ActivatedRoute = inject(ActivatedRoute);
  private profilesService: ProfilesService = inject(ProfilesService);

  ngOnInit() {
    this.profileId = this.route.snapshot.params['profileId'];

    if (this.profileId) {
      this.profilesService.getIcon(this.profileId).subscribe({
        next: (icon: Icon | null) => {
          if (icon) {
            this.type = 'Edit';
          } else {
            this.type = 'Add';
          }
        },
      });
    }
  }

  onSelectedFile(event: any) {
    this.processFile(event.target.files[0]);
  }

  onDragOver(event: DragEvent) {
    event.preventDefault();
    this.isDragOver = true;
  }

  onDragLeave(event: DragEvent) {
    event.preventDefault();
    this.isDragOver = false;
  }

  onDrop(event: DragEvent) {
    event.preventDefault();
    this.isDragOver = false;
    if (event.dataTransfer?.files.length) {
      this.processFile(event.dataTransfer.files[0]);
    }
  }

  processFile(file: File) {
    this.errorMessage = '';

    if (!this.isImage(file)) {
      this.errorMessage = 'Invalid file type. Please upload an image.';
      return;
    }

    if (file.size > this.maxFileSize) {
      this.errorMessage = 'File size exceeds 2MB limit.';
      return;
    }

    this.setFile(file);
  }

  isImage(file: File): boolean {
    return file.type.startsWith('image/');
  }

  setFile(file: File) {
    this.selectedFile = Object.assign(file, {
      objectURL: URL.createObjectURL(file),
    });
  }

  removeFile(event: Event) {
    event.stopPropagation();
    this.selectedFile = null;
  }

  formatSize(size: number): string {
    return size < 1024
      ? `${size} B`
      : size < 1048576
        ? `${(size / 1024).toFixed(1)} KB`
        : `${(size / 1048576).toFixed(1)} MB`;
  }

  onSubmit() {
    if (this.selectedFile && this.profileId) {
      if (this.type == 'Add') {
        this.profilesService
          .addIcon(this.profileId, this.selectedFile)
          .subscribe({
            next: (response) => {
              this.router.navigate([`../../../user/${response.userId}`], {
                relativeTo: this.route,
              });
            },
          });
      } else {
        this.profilesService
          .editIcon(this.profileId, this.selectedFile)
          .subscribe({
            next: (response) => {
              this.router.navigate([`../../../user/${response.userId}`], {
                relativeTo: this.route,
              });
            },
          });
      }
    }
  }

  onReset() {
    this.selectedFile = null;
    this.errorMessage = '';

    const fileInput = document.querySelector(
      'input[type="file"]',
    ) as HTMLInputElement;
    if (fileInput) {
      fileInput.value = '';
    }
  }
}
