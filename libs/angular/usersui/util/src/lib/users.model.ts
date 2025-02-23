import { UserRole } from '@jdw/angular-shared-util';

export type Address = {
  id: number;
  addressLine1: string;
  addressLine2: string;
  city: string;
  stateProvince: string;
  postalCode: string;
  country: string;
  profileId: number;
  createdByUserId: number;
  createdTime: string;
  modifiedByUserId: number;
  modifiedTime: string;
};

export type AddressRequest = {
  addressLine1: string;
  addressLine2: string;
  city: string;
  stateProvince: string;
  postalCode: string;
  country: string;
};

export type Icon = {
  id: number;
  profileId: number;
  icon: string;
  createdByUserId: number;
  createdTime: string;
  modifiedByUserId: number;
  modifiedTime: string;
};

export type Profile = {
  id: number;
  firstName: string;
  middleName: string | null;
  lastName: string;
  birthdate: string;
  userId: number;
  addresses: Address[];
  icon: Icon | null;
  createdByUserId: number;
  createdTime: string;
  modifiedByUserId: number;
  modifiedTime: string;
};

export type AddProfile = {
  firstName: string;
  middleName: string;
  lastName: string;
  birthdate: string;
  userId: number;
};

export type EditProfile = {
  firstName: string;
  middleName: string;
  lastName: string;
  birthdate: string;
};

export type User = {
  id: number;
  emailAddress: string;
  password: string;
  status: string;
  roles: UserRole[];
  profile: Profile | null;
  createdByUserId: number;
  createdTime: string;
  modifiedByUserId: number;
  modifiedTime: string;
};
