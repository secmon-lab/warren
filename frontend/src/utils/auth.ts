import { ANONYMOUS_USER_ID } from '../constants/auth';

export interface User {
  sub: string;
  email: string;
  name: string;
}

export const isAnonymousUser = (user: User): boolean => {
  return user.sub === ANONYMOUS_USER_ID;
};