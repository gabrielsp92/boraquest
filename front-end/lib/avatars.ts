// Fallback icon for a member id that isn't in the guild member list (e.g. before it
// loads, or a former member still referenced by an entry). Members in the list render
// with the avatarKey GET /guild/members returns for them.
export const DEFAULT_AVATAR = "/avatars/default.svg";

// The platform's six avatar icons the signed-in member can pick for themself. The
// backend only ever stores one of these keys (avatarKey) against a user — this file
// is the only place that knows the key maps to an actual asset.
export type AvatarKey = "lia" | "beto" | "nena" | "caio" | "duda" | "tom";

/** Fixed display order in the picker grid; also the "Avatar N" aria-label order. */
export const AVATAR_KEYS: AvatarKey[] = ["lia", "beto", "nena", "caio", "duda", "tom"];

export const avatarSrc = (key: string): string => `/avatars/${key}.svg`;

/** A guild member's avatar, or DEFAULT_AVATAR when the member is unknown. */
export const memberAvatarSrc = (member: { avatarKey: string } | undefined): string =>
  member ? avatarSrc(member.avatarKey) : DEFAULT_AVATAR;
