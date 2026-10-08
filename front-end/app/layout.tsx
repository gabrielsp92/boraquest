import type { Metadata, Viewport } from "next";
import localFont from "next/font/local";
import "./globals.css";

// Baloo 2 and Nunito (both SIL Open Font License), self-hosted so the PWA works offline.
const baloo = localFont({
  variable: "--font-baloo",
  src: [
    { path: "./fonts/baloo-2-latin-700-normal.woff2", weight: "700" },
    { path: "./fonts/baloo-2-latin-800-normal.woff2", weight: "800" },
  ],
});
const nunito = localFont({
  variable: "--font-nunito",
  src: [
    { path: "./fonts/nunito-latin-700-normal.woff2", weight: "700" },
    { path: "./fonts/nunito-latin-800-normal.woff2", weight: "800" },
  ],
});

export const metadata: Metadata = {
  title: "Boraquest",
  description: "O jogo de hábitos da família.",
  appleWebApp: { capable: true, title: "Boraquest", statusBarStyle: "default" },
};

export const viewport: Viewport = {
  themeColor: "#fff8ec",
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="pt-BR" className={`${baloo.variable} ${nunito.variable}`}>
      <body>{children}</body>
    </html>
  );
}
