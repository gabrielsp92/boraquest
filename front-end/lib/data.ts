// Sample data so the starter runs without a backend. Replace with your API or database.

export type MemberId = "lia" | "beto" | "nena" | "caio";
export type Member = { id: MemberId; name: string; avatar: string };
export type Frequency = "diaria" | "semanal";
export type Quest = { id: string; name: string; short: string; frequency: Frequency; points: number };

export const members: Record<MemberId, Member> = {
  lia: { id: "lia", name: "Lia", avatar: "/avatars/lia.svg" },
  beto: { id: "beto", name: "Beto", avatar: "/avatars/beto.svg" },
  nena: { id: "nena", name: "Vó Nena", avatar: "/avatars/nena.svg" },
  caio: { id: "caio", name: "Caio", avatar: "/avatars/caio.svg" },
};
export const memberList = Object.values(members);

/** The signed-in member. Swap for real auth. */
export const me: MemberId = "lia";

export const quests: Quest[] = [
  { id: "agua", name: "Beber 2 L de água", short: "Água", frequency: "diaria", points: 10 },
  { id: "caminhada", name: "Caminhar 30 minutos", short: "Caminhada", frequency: "diaria", points: 15 },
  { id: "leitura", name: "Ler 20 minutos", short: "Leitura", frequency: "diaria", points: 10 },
  { id: "sono", name: "Dormir antes das 23h", short: "Sono", frequency: "diaria", points: 10 },
  { id: "treino", name: "Treinar 3 vezes", short: "Treino", frequency: "semanal", points: 30 },
  { id: "celular", name: "Celular na mesa", short: "Celular na mesa", frequency: "diaria", points: -5 },
  { id: "cafe", name: "Pular o café da manhã", short: "Sem café da manhã", frequency: "diaria", points: -5 },
  { id: "fastfood", name: "Comer fast-food", short: "Fast-food", frequency: "diaria", points: -10 },
];
export const questById = (id: string) => quests.find((q) => q.id === id)!;

export const prizes = { week: "Escolher o filme de sábado", month: "Jantar no restaurante favorito" };

export const today = { label: "Quinta, 8 de outubro", week: "5 a 11 de outubro" };

/** What the signed-in member has done today. */
export const todayDone: { questId: string; photo?: string }[] = [
  { questId: "agua", photo: "placeholder" },
  { questId: "caminhada" },
  { questId: "leitura" },
];
export const todaySlips: { questId: string; by: MemberId }[] = [{ questId: "celular", by: "beto" }];

export type Day = {
  label: string;
  done: string[];
  slips: { questId: string; by: MemberId }[];
  photos: number;
};
export const week: Day[] = [
  { label: "Hoje · quinta", done: ["agua", "caminhada", "leitura"], slips: [{ questId: "celular", by: "beto" }], photos: 1 },
  { label: "Quarta", done: ["agua", "caminhada", "sono"], slips: [], photos: 2 },
  { label: "Terça", done: ["agua", "caminhada"], slips: [{ questId: "celular", by: "nena" }], photos: 0 },
];

export const pastWeeks: { range: string; winner: MemberId; points: number; prize: string }[] = [
  { range: "28 de set. a 4 de out.", winner: "nena", points: 140, prize: "Escolher o filme de sábado" },
  { range: "21 a 27 de setembro", winner: "lia", points: 125, prize: "Café da manhã na cama" },
  { range: "14 a 20 de setembro", winner: "nena", points: 130, prize: "Escolher o filme de sábado" },
  { range: "7 a 13 de setembro", winner: "beto", points: 110, prize: "Folga da louça no domingo" },
];

export type Standing = { member: MemberId; points: number; auditPending?: boolean };
export const weekStandings: Standing[] = [
  { member: "nena", points: 120 },
  { member: "lia", points: 85 },
  { member: "caio", points: 80, auditPending: true },
  { member: "beto", points: 60 },
];
export const monthStandings: Standing[] = [
  { member: "nena", points: 410 },
  { member: "lia", points: 395 },
  { member: "beto", points: 330 },
  { member: "caio", points: 300 },
];

export type ProofItem = { questId: string; day: string; photo?: boolean; note?: string };
export const auditProofs: ProofItem[] = [
  { questId: "caminhada", day: "Terça", photo: true },
  { questId: "agua", day: "Quarta", note: "Enchi a garrafa azul duas vezes." },
  { questId: "leitura", day: "Quarta" },
];

export const sum = (ids: string[]) => ids.reduce((n, id) => n + questById(id).points, 0);
