import { Component, ChangeDetectionStrategy } from '@angular/core';

import { ReactiveFormsModule } from '@angular/forms';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import {
  MatCard,
  MatCardActions,
  MatCardContent,
  MatCardTitle,
} from '@angular/material/card';
import { RouterLink } from '@angular/router';
import { EmailSignInComponent } from '../email-sign-in/email-sign-in.component';

@Component({
  selector: 'lib-sign-in',
  imports: [
    MatButtonModule,
    MatInputModule,
    MatIconModule,
    ReactiveFormsModule,
    MatCardActions,
    RouterLink,
    EmailSignInComponent,
    MatCardTitle,
    MatCardContent,
    MatCard,
  ],
  templateUrl: './sign-in.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './sign-in.component.scss',
})
export class SignInComponent {}
