// EXPO_PUBLIC_* variables are inlined at build time. On a phone, localhost is
// the phone itself: point this at the machine's LAN IP (see .env.example).
export const API_URL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';

// Operator logos carry a trademark risk (see Notion, Design e UI): keep them
// switchable so they can be removed quickly.
export const SHOW_OPERATOR_LOGO = true;
