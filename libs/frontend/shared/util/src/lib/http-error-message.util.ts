import { HttpErrorResponse } from '@angular/common/http';

export function getErrorMessage(error: HttpErrorResponse): string {
  switch (error.status) {
    case 400:
      return 'There was an issue with your submission. Please check your input and try again.';
    case 401:
      return 'Your session has expired. Please log in again to continue.';
    case 403:
      return 'You do not have the necessary permissions to perform this action.';
    case 404:
      return 'The requested resource could not be found. Please verify the URL or resource and try again.';
    case 409:
      return 'A conflict occurred. The record you are trying to create already exists.';
    case 500:
      return 'An unexpected error occurred on our server. Please try again later.';
    default:
      return `An error occurred. Please contact support if the issue persists.`;
  }
}
