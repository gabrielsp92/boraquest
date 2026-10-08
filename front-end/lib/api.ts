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

export const login = (email: string, password: string) => apiFetch<Session>("/auth/login", "POST", { email, password });

export const listRules = () => apiFetch<RuleList>("/rules");
export const createRule = (input: RuleInput) => apiFetch<Rule>("/rules", "POST", input);
export const updateRule = (id: string, input: RuleInput) => apiFetch<Rule>(`/rules/${encodeURIComponent(id)}`, "PUT", input);
export const deleteRule = (id: string) => apiFetch<void>(`/rules/${encodeURIComponent(id)}`, "DELETE");

/** Signed points a rule is worth: positive for sum, negative for decrease. */
export const rulePoints = (r: Pick<Rule, "scoreType" | "score">) => (r.scoreType === "sum" ? r.score : -r.score);
