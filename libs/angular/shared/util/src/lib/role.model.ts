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

export type UserRole = {
  userId: number;
  roleId: number;
  createdByUserId: number;
  createdTime: string;
};
