import { UserRole } from '@jdw/angular-usersui-util';

export type Role = {
  id: number;
  name: string;
  description: string;
  status: string;
  users: UserRole[];
  createdByUserId: number;
  createdTime: string;
  modifiedByUserId: number;
  modifiedTime: string;
};
