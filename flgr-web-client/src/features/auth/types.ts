// The User shape itself lives in src/types/user.ts (shared across
// features — e.g. AppLayout and HomePage also display it), not here.
export interface LoginRequest {
  email: string
  password: string
}
