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

export type AddRole = {
  name: string;
  description: string;
};

export type EditRole = {
  name: string;
  description: string;
};
