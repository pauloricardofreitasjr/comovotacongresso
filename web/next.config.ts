import type { NextConfig } from "next";

const config: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  // As fotos vêm direto do portal da Câmara (ver docs/discovery §3.3).
  images: { unoptimized: true },
};

export default config;
