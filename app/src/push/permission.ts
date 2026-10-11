import AsyncStorage from '@react-native-async-storage/async-storage';
import { PermissionsAndroid, Platform } from 'react-native';

export type PermissionState = 'granted' | 'denied' | 'undetermined';

// Android only reports granted or not; remembering the refusal tells
// "never asked" (explain and ask) apart from "refused" (point to settings).
const DENIED_KEY = 'notificationsDenied';

// Android 13 (API 33) made notifications a runtime permission.
const asksAtRuntime = () => Platform.OS === 'android' && Number(Platform.Version) >= 33;

export async function notificationPermission(): Promise<PermissionState> {
  if (!asksAtRuntime()) return 'granted';
  if (await PermissionsAndroid.check(PermissionsAndroid.PERMISSIONS.POST_NOTIFICATIONS)) return 'granted';
  return (await AsyncStorage.getItem(DENIED_KEY)) ? 'denied' : 'undetermined';
}

export async function requestNotificationPermission(): Promise<'granted' | 'denied'> {
  if (!asksAtRuntime()) return 'granted';
  const answer = await PermissionsAndroid.request(PermissionsAndroid.PERMISSIONS.POST_NOTIFICATIONS);
  if (answer === PermissionsAndroid.RESULTS.GRANTED) return 'granted';
  await AsyncStorage.setItem(DENIED_KEY, '1');
  return 'denied';
}
