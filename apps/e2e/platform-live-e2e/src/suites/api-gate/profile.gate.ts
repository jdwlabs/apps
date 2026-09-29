import { test, expect } from '../../harness/fixtures';
import { operations } from '../../harness/operations';
import { ADMIN, GATE, PRD_SAFE } from '../../harness/tags';
import type { EphemeralUser } from '../../harness/fixtures';
import { seededCredentials } from '../../harness/test-users';

// 1x1 transparent PNG.
const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
  'base64',
);

function iconForm(): FormData {
  const form = new FormData();
  form.append('icon', new Blob([PNG], { type: 'image/png' }), 'icon.png');
  return form;
}

const profileBody = {
  firstName: 'Live',
  lastName: 'Gate',
  birthdate: '1990-01-01T00:00:00Z',
};

async function createOwnProfile(user: EphemeralUser): Promise<number> {
  const created = await user.clients.profile.POST('/api/profiles', {
    body: { ...profileBody, userId: user.id },
  });
  expect(created.response.status).toBe(201);
  return created.data?.id as number;
}

test.describe('profile', () => {
  test(
    'seeded user reads its own profile',
    {
      tag: [GATE, PRD_SAFE],
      annotation: operations('getProfileByUserId', 'getProfileById'),
    },
    async ({ asUser }) => {
      // The seeded account must own a profile; phase 0 creates it.
      const { emailAddress } = seededCredentials(process.env, 'user');
      const self = await asUser.identity.GET(
        '/api/users/email/{emailAddress}',
        {
          params: { path: { emailAddress } },
        },
      );
      const userId = self.data?.id as number;
      const byUser = await asUser.profile.GET(
        '/api/profiles/by-user/{userId}',
        {
          params: { path: { userId } },
        },
      );
      expect(byUser.response.status).toBe(200);
      const profileId = byUser.data?.id as number;
      const byId = await asUser.profile.GET('/api/profiles/{profileId}', {
        params: { path: { profileId } },
      });
      expect(byId.response.status).toBe(200);
      expect(byId.data?.userId).toBe(userId);
    },
  );

  test(
    'a user manages its profile, address and icon by profile id',
    {
      tag: [GATE],
      annotation: operations(
        'createProfile',
        'getProfileById',
        'updateProfileById',
        'addAddress',
        'updateAddress',
        'deleteAddress',
        'addIcon',
        'getProfileIcon',
        'updateIcon',
        'deleteIcon',
        'deleteProfileById',
      ),
    },
    async ({ newEphemeralUser }) => {
      const user = await newEphemeralUser();
      const profileId = await createOwnProfile(user);
      const path = { params: { path: { profileId } } };
      const api = user.clients.profile;

      expect(
        (await api.GET('/api/profiles/{profileId}', path)).data?.firstName,
      ).toBe('Live');
      expect(
        (
          await api.PUT('/api/profiles/{profileId}', {
            ...path,
            body: { ...profileBody, firstName: 'Updated' },
          })
        ).response.status,
      ).toBe(200);

      const address = {
        addressLine1: '1 Test St',
        city: 'Testville',
        stateProvince: 'TS',
        postalCode: '00000',
        country: 'US',
      };
      const added = await api.POST('/api/profiles/{profileId}/address', {
        ...path,
        body: address,
      });
      expect(added.response.status).toBe(200);
      const addressId = added.data?.addresses?.[0]?.id as number;
      expect(addressId).toBeDefined();
      const addressPath = { params: { path: { profileId, addressId } } };
      expect(
        (
          await api.PUT('/api/profiles/{profileId}/address/{addressId}', {
            ...addressPath,
            body: { ...address, city: 'Elsewhere' },
          })
        ).response.status,
      ).toBe(200);
      expect(
        (
          await api.DELETE(
            '/api/profiles/{profileId}/address/{addressId}',
            addressPath,
          )
        ).response.status,
      ).toBe(204);

      // IconUpload's binary field is typed as a string; the serializer replaces
      // the placeholder body with the real multipart form.
      const upload = {
        ...path,
        body: { icon: '' },
        bodySerializer: () => iconForm(),
      };
      expect(
        (await api.POST('/api/profiles/{profileId}/icon', upload)).response
          .status,
      ).toBe(200);
      const icon = await api.GET('/api/profiles/{profileId}/icon', {
        ...path,
        parseAs: 'arrayBuffer',
      });
      expect(icon.response.status).toBe(200);
      expect(icon.response.headers.get('content-type')).toContain('image/png');
      expect(Buffer.from(icon.data as ArrayBuffer).subarray(0, 4)).toEqual(
        PNG.subarray(0, 4),
      );
      expect(
        (await api.PUT('/api/profiles/{profileId}/icon', upload)).response
          .status,
      ).toBe(200);
      expect(
        (await api.DELETE('/api/profiles/{profileId}/icon', path)).response
          .status,
      ).toBe(204);

      expect(
        (await api.DELETE('/api/profiles/{profileId}', path)).response.status,
      ).toBe(204);
    },
  );

  test(
    'a user manages its profile by user id',
    {
      tag: [GATE],
      annotation: operations(
        'createProfile',
        'getProfileByUserId',
        'updateProfileByUserId',
        'deleteProfileByUserId',
      ),
    },
    async ({ newEphemeralUser }) => {
      const user = await newEphemeralUser();
      await createOwnProfile(user);
      const path = { params: { path: { userId: user.id } } };
      const api = user.clients.profile;
      expect(
        (await api.GET('/api/profiles/by-user/{userId}', path)).data?.userId,
      ).toBe(user.id);
      expect(
        (
          await api.PUT('/api/profiles/by-user/{userId}', {
            ...path,
            body: { ...profileBody, lastName: 'Changed' },
          })
        ).response.status,
      ).toBe(200);
      expect(
        (await api.DELETE('/api/profiles/by-user/{userId}', path)).response
          .status,
      ).toBe(204);
    },
  );

  test(
    'admin lists profiles',
    { tag: [GATE, ADMIN], annotation: operations('getProfiles') },
    async ({ asAdmin }) => {
      const all = await asAdmin.profile.GET('/api/profiles');
      expect(all.response.status).toBe(200);
      expect(Array.isArray(all.data)).toBe(true);
    },
  );
});
