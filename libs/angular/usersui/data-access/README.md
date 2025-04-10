# Angular UsersUI Data Access

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

The **Angular UsersUI Data Access** library provides services to manage and retrieve user and profile data for the Users
UI module within the JDW platform. It encapsulates API communication and business logic related to user and profile
operations, promoting reuse and consistency across the application.

---

## 📁 Project Structure

```
libs/angular/usersui/data-access/
├── src
│   ├── index.ts                       # Public API export for the library
│   ├── lib
│   │   ├── profiles
│   │   │   ├── profiles.service.ts    # Service for managing profile data
│   │   │   └── profiles.service.spec.ts
│   │   └── users
│   │       ├── users.service.ts       # Service for managing user data
│   │       └── users.service.spec.ts
│   └── test-setup.ts                  # Test setup configuration
├── jest.config.ts                     # Jest configuration for unit testing
├── tsconfig.json                      # Base TypeScript configuration
├── tsconfig.lib.json                  # Library-specific TypeScript configuration
└── tsconfig.spec.json                 # Test-specific TypeScript configuration
```

---

## 🔧 Usage

### Importing a Service

You can import the services provided by this library directly into your Angular modules or components. For example, to
use the UsersService:

```ts
import { UsersService } from '@jdw/angular-usersui-data-access';

@Component({
  selector: 'app-my-component',
  template: `<p>My Component Content</p>`
})
export class MyComponent {
  private usersService = inject(UsersService);

  ngOnInit(): void {
    this.usersService.getUsers().subscribe(users => {
      console.log(users);
    });
  }
}
```

### Using the ProfilesService

```ts
import { ProfilesService } from '@jdw/angular-usersui-data-access';

@Injectable({ providedIn: 'root' })
export class ProfileManager {
  private profilesService = inject(ProfilesService);

  loadProfiles(): void {
    this.profilesService.getProfiles().subscribe(profiles => {
      console.log(profiles);
    });
  }
}
```

---

## 🧪 Testing

Run unit tests for the library with:

```bash
nx test angular-usersui-data-access
```

---

## 🛠 Linting

To check code quality with ESLint:

```bash
nx lint angular-usersui-data-access
```

---

## 📌 Notes

- This library centralizes data access logic for Users UI and is designed for reuse in feature and container
  applications.
- It follows Angular best practices and supports strict typing and unit testing.
- The library is maintained as part of a modular Nx monorepo, ensuring consistency across related packages.

---

## 📚 Related Packages

- [`@jdw/angular-usersui-feature-core`](../feature/core): Contains core UI components for Users UI.
- [`@jdw/angular-usersui-util`](../util): Provides shared utilities and models for Users UI.
