import { Linking } from 'react-native';

import { notificationPermission, requestNotificationPermission, type PermissionState } from './permission';
import { subscribeLine, unsubscribeLine } from './topics';

export type { PermissionState };

// What the screen needs from push; tests pass a fake.
export type Push = {
  subscribe: (code: number) => Promise<void>;
  unsubscribe: (code: number) => Promise<void>;
  permission: () => Promise<PermissionState>;
  requestPermission: () => Promise<'granted' | 'denied'>;
  openSettings: () => void;
};

export const firebasePush: Push = {
  subscribe: subscribeLine,
  unsubscribe: unsubscribeLine,
  permission: notificationPermission,
  requestPermission: requestNotificationPermission,
  openSettings: () => Linking.openSettings(),
};
