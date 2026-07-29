import { JwtPayload } from 'jwt-decode';

export interface DecodedToken extends JwtPayload {
  user_id?: string | number;
  profile_id?: string | number;
  roles?: string[];
}
