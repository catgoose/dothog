// setup:feature:capacitor
import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.catgoose.dothog',
  appName: 'Dothog',
  webDir: 'build/web/assets/public',
  server: {
    // In development, point at the local Go server.
    // For production builds, comment this out — Capacitor will serve
    // the bundled webDir files directly.
    url: 'https://localhost:3000',
    cleartext: false,
  },
};

export default config;
