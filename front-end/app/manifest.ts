import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Boraquest",
    short_name: "Boraquest",
    description: "O jogo de hábitos da família.",
    lang: "pt-BR",
    start_url: "/hoje",
    display: "standalone",
    orientation: "portrait",
    background_color: "#fff8ec",
    theme_color: "#ffb300",
    icons: [
      { src: "/icons/icon-192.png", sizes: "192x192", type: "image/png" },
      { src: "/icons/icon-512.png", sizes: "512x512", type: "image/png" },
      { src: "/icons/icon-maskable-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
    ],
  };
}
