// Placeholder avatar for other guild members fetched from the backend: there is no
// "list guild members" avatar broadcast yet (see todo/profile-avatar.md's Non-goals),
// so everyone but the signed-in member still renders with this one generic icon.
export const DEFAULT_AVATAR = "/avatars/default.svg";

// The platform's six avatar icons the signed-in member can pick for themself. The
// backend only ever stores one of these keys (avatarKey) against a user — this file
// is the only place that knows the key maps to an actual asset.
export type AvatarKey = "lia" | "beto" | "nena" | "caio" | "duda" | "tom";

/** Fixed display order in the picker grid; also the "Avatar N" aria-label order. */
export const AVATAR_KEYS: AvatarKey[] = ["lia", "beto", "nena", "caio", "duda", "tom"];

export const avatarSrc = (key: string): string => `/avatars/${key}.svg`;
