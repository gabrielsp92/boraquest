// Client for the Boraquest API (back-end/api/openapi.yaml). Calls go through the
// /api/v1 rewrite in next.config.ts.
import { clearSession, getSession, type Session } from "./auth";

export type RuleFrequency = "daily" | "weekly";
export type ScoreType = "sum" | "decrease";
export type Rule = {
  id: string;
  name: string;
  frequency: RuleFrequency;
  scoreType: ScoreType;
  score: number;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
};
export type RuleInput = Pick<Rule, "name" | "frequency" | "scoreType" | "score">;
export type RuleList = { rules: Rule[]; limit: number };

export type Prizes = { week: string; month: string; updatedAt: string };
export type PrizePeriod = "week" | "month";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function apiFetch<T>(path: string, method = "GET", body?: unknown): Promise<T> {
  const session = getSession();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (session) headers.Authorization = `Bearer ${session.token}`;

  const res = await fetch(`/api/v1${path}`, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  // An expired or revoked token signs the user out; AuthGate then sends them to /entrar.
  if (res.status === 401 && session) clearSession();
  if (!res.ok) {
    const data = (await res.json().catch(() => null)) as { error?: string } | null;
    throw new ApiError(res.status, data?.error ?? res.statusText);
  }
  return (res.status === 204 ? undefined : await res.json()) as T;
}

async function apiFetchForm<T>(path: string, form: FormData): Promise<T> {
  const session = getSession();
  const headers: Record<string, string> = {};
  if (session) headers.Authorization = `Bearer ${session.token}`;

  // No Content-Type: the browser sets it, including the multipart boundary.
  const res = await fetch(`/api/v1${path}`, { method: "POST", headers, body: form });
  if (res.status === 401 && session) clearSession();
  if (!res.ok) {
    const data = (await res.json().catch(() => null)) as { error?: string } | null;
    throw new ApiError(res.status, data?.error ?? res.statusText);
  }
  return (await res.json()) as T;
}

export const login = (email: string, password: string) => apiFetch<Session>("/auth/login", "POST", { email, password });

export const updateMyAvatar = (avatarKey: string) =>
  apiFetch<{ id: string; name: string; email: string; avatarKey: string }>("/me/avatar", "PUT", { avatarKey });

export const listRules = () => apiFetch<RuleList>("/rules");
export const createRule = (input: RuleInput) => apiFetch<Rule>("/rules", "POST", input);
export const updateRule = (id: string, input: RuleInput) => apiFetch<Rule>(`/rules/${encodeURIComponent(id)}`, "PUT", input);
export const deleteRule = (id: string) => apiFetch<void>(`/rules/${encodeURIComponent(id)}`, "DELETE");

/** Signed points a rule is worth: positive for sum, negative for decrease. */
export const rulePoints = (r: Pick<Rule, "scoreType" | "score">) => (r.scoreType === "sum" ? r.score : -r.score);

export const getPrizes = () => apiFetch<Prizes>("/prizes");
export const setPrize = (period: PrizePeriod, text: string) => apiFetch<Prizes>(`/prizes/${period}`, "PUT", { text });

export type EntryPeriod = "today" | "week";
export type Entry = {
  id: string;
  ruleId: string;
  ruleName: string;
  scoreType: ScoreType;
  points: number; // signed
  memberId: string;
  loggedBy: string;
  occurredOn: string; // "YYYY-MM-DD"
  createdAt: string;
  photoUrl: string; // "" when the entry has no proof photo
};
export type EntryList = { entries: Entry[]; from: string; to: string; today: string }; // "YYYY-MM-DD"

export const listEntries = (period: EntryPeriod) => apiFetch<EntryList>(`/entries?period=${period}`);
export const createEntry = (ruleId: string, memberId: string) =>
  apiFetch<Entry>("/entries", "POST", { ruleId, memberId });
export const deleteEntry = (id: string) => apiFetch<void>(`/entries/${encodeURIComponent(id)}`, "DELETE");

/** Limits the API enforces on a proof photo (back-end entry.ValidPhotoFile). */
export const PHOTO_TYPES = ["image/jpeg", "image/png", "image/webp"];
export const PHOTO_MAX_BYTES = 8 * 1024 * 1024;

export const uploadEntryPhoto = (id: string, file: File) => {
  const form = new FormData();
  form.append("photo", file);
  return apiFetchForm<Entry>(`/entries/${encodeURIComponent(id)}/photo`, form);
};

export type GuildMember = { id: string; name: string };
export type GuildMemberList = { members: GuildMember[] };
export const listGuildMembers = () => apiFetch<GuildMemberList>("/guild/members");

export type ScoreboardPeriod = "week" | "month";
export type Standing = { memberId: string; points: number; completed: number };
export type Scoreboard = { standings: Standing[]; from: string; to: string; today: string };

export const getScoreboard = (period: ScoreboardPeriod) => apiFetch<Scoreboard>(`/scoreboard?period=${period}`);
