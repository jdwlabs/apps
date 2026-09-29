import { test, expect } from '../../harness/fixtures';
import { operations } from '../../harness/operations';
import { ADMIN, GATE, PRD_SAFE } from '../../harness/tags';
import {
  ephemeralEmail,
  generatePassword,
  isGoneOrDeleted,
  seededCredentials,
} from '../../harness/test-users';

test.describe('identity', () => {
  test(
    'seeded user authenticates and reads itself',
    {
      tag: [GATE, PRD_SAFE],
      annotation: operations(
        'authenticate',
        'getUserById',
        'getUserByEmailAddress',
      ),
    },
    async ({ asUser }) => {
      const { emailAddress } = seededCredentials(process.env, 'user');
      const byEmail = await asUser.identity.GET(
        '/api/users/email/{emailAddress}',
        {
          params: { path: { emailAddress } },
        },
      );
      expect(byEmail.response.status).toBe(200);
      const id = byEmail.data?.id;
      expect(id).toBeDefined();
      const byId = await asUser.identity.GET('/api/users/{userId}', {
        params: { path: { userId: id as number } },
      });
      expect(byId.response.status).toBe(200);
      expect(byId.data?.emailAddress).toBe(emailAddress);
    },
  );

  test(
    'a new user registers, signs in, updates and deletes itself',
    {
      tag: [GATE],
      annotation: operations(
        'registerUser',
        'authenticate',
        'updateUser',
        'deleteUser',
      ),
    },
    async ({ newEphemeralUser }) => {
      const user = await newEphemeralUser();
      const updated = await user.clients.identity.PUT('/api/users/{userId}', {
        params: { path: { userId: user.id } },
        body: {
          emailAddress: user.credentials.emailAddress,
          password: generatePassword(),
        },
      });
      expect(updated.response.status).toBe(200);
      const deleted = await user.clients.identity.DELETE(
        '/api/users/{userId}',
        {
          params: { path: { userId: user.id } },
        },
      );
      expect(deleted.response.status).toBe(204);
    },
  );

  test(
    'admin lists and creates users',
    {
      tag: [GATE, ADMIN],
      annotation: operations('getAllUsers', 'createUser'),
    },
    async ({ asAdmin, runId }, testInfo) => {
      const all = await asAdmin.identity.GET('/api/users');
      expect(all.response.status).toBe(200);
      expect(Array.isArray(all.data)).toBe(true);

      const created = await asAdmin.identity.POST('/api/users', {
        body: {
          emailAddress: ephemeralEmail(runId, testInfo.workerIndex, 900),
          password: generatePassword(),
        },
      });
      expect(created.response.status).toBe(201);
      const id = created.data?.id;
      expect(id).toBeDefined();
      let bodySucceeded = false;
      try {
        const byId = await asAdmin.identity.GET('/api/users/{userId}', {
          params: { path: { userId: id as number } },
        });
        expect(byId.response.status).toBe(200);
        bodySucceeded = true;
      } finally {
        const cleanup = await asAdmin.identity.DELETE('/api/users/{userId}', {
          params: { path: { userId: id as number } },
        });
        if (bodySucceeded) {
          expect(isGoneOrDeleted(cleanup.response.status)).toBe(true);
        }
      }
    },
  );

  test(
    'admin runs a role through its whole lifecycle',
    {
      tag: [GATE, ADMIN],
      annotation: operations(
        'createRole',
        'getAllRoles',
        'getRoleById',
        'getRoleByName',
        'updateRole',
        'grantUsersToRole',
        'revokeUsersFromRole',
        'grantRolesToUser',
        'revokeRolesFromUser',
        'deleteRole',
      ),
    },
    async ({ asAdmin, newEphemeralUser, runId }, testInfo) => {
      const member = await newEphemeralUser();
      const name = `e2e-${runId}-w${testInfo.workerIndex}`;

      const created = await asAdmin.identity.POST('/api/roles', {
        body: { name, description: 'live e2e role' },
      });
      expect(created.response.status).toBe(201);
      const roleId = created.data?.id as number;

      let bodySucceeded = false;
      try {
        expect((await asAdmin.identity.GET('/api/roles')).response.status).toBe(
          200,
        );
        expect(
          (
            await asAdmin.identity.GET('/api/roles/{roleId}', {
              params: { path: { roleId } },
            })
          ).data?.name,
        ).toBe(name);
        expect(
          (
            await asAdmin.identity.GET('/api/roles/name/{roleName}', {
              params: { path: { roleName: name } },
            })
          ).data?.id,
        ).toBe(roleId);

        const updated = await asAdmin.identity.PUT('/api/roles/{roleId}', {
          params: { path: { roleId } },
          body: { name, description: 'live e2e role, updated' },
        });
        expect(updated.response.status).toBe(200);

        const path = { params: { path: { roleId } } };
        expect(
          (
            await asAdmin.identity.PUT('/api/roles/{roleId}/users/grant', {
              ...path,
              body: [member.id],
            })
          ).response.status,
        ).toBe(200);
        expect(
          (
            await asAdmin.identity.PUT('/api/roles/{roleId}/users/revoke', {
              ...path,
              body: [member.id],
            })
          ).response.status,
        ).toBe(200);

        const userPath = { params: { path: { userId: member.id } } };
        expect(
          (
            await asAdmin.identity.PUT('/api/users/{userId}/roles/grant', {
              ...userPath,
              body: [roleId],
            })
          ).response.status,
        ).toBe(200);
        expect(
          (
            await asAdmin.identity.PUT('/api/users/{userId}/roles/revoke', {
              ...userPath,
              body: [roleId],
            })
          ).response.status,
        ).toBe(200);
        bodySucceeded = true;
      } finally {
        const deleted = await asAdmin.identity.DELETE('/api/roles/{roleId}', {
          params: { path: { roleId } },
        });
        // Asserting here while the body is failing would replace its error.
        if (bodySucceeded) {
          expect(deleted.response.status).toBe(204);
        } else if (deleted.response.status !== 204) {
          console.warn(
            `Role ${roleId} cleanup returned HTTP ${deleted.response.status}`,
          );
        }
      }
    },
  );
});
