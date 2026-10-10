// pt-BR day/week labels for Hoje and Semana, anchored to the guild's fixed day
// boundary (America/Sao_Paulo) regardless of the viewing device's timezone.
// Entries arrive as "YYYY-MM-DD" date-only strings; every Date built here is pinned
// to UTC noon so formatting can never shift the calendar day in either direction.
//
// pt-BR weekday names are the colloquial short form used throughout the design
// ("quinta", not "quinta-feira") — Intl's {weekday:"long"} would produce the
// formal "-feira" form, so weekdays are a hardcoded lookup instead.

const WEEKDAYS = ["domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"]; // Date#getUTCDay() order
const DAY_MONTH = new Intl.DateTimeFormat("pt-BR", { day: "numeric", month: "long" });

function parseDateOnly(iso: string): Date {
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d, 12));
}

/** "Hoje · quinta" when isToday, else "Quarta" / "Terça" (capitalized). */
export function dayLabel(iso: string, isToday: boolean): string {
  const weekday = WEEKDAYS[parseDateOnly(iso).getUTCDay()];
  return isToday ? `Hoje · ${weekday}` : weekday[0].toUpperCase() + weekday.slice(1);
}

/** "Quinta, 8 de outubro" for Hoje's TopBar caption. */
export function todayCaption(iso: string): string {
  const date = parseDateOnly(iso);
  const weekday = WEEKDAYS[date.getUTCDay()];
  return `${weekday[0].toUpperCase()}${weekday.slice(1)}, ${DAY_MONTH.format(date)}`;
}

/** "5 a 11 de outubro" (same month) or "28 de setembro a 4 de outubro" (crosses months). */
export function weekRangeLabel(fromIso: string, toIso: string): string {
  const from = parseDateOnly(fromIso);
  const to = parseDateOnly(toIso);
  const sameMonth = from.getUTCMonth() === to.getUTCMonth() && from.getUTCFullYear() === to.getUTCFullYear();
  const toLabel = DAY_MONTH.format(to);
  return sameMonth ? `${from.getUTCDate()} a ${toLabel}` : `${DAY_MONTH.format(from)} a ${toLabel}`;
}
