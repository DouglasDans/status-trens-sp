import { getApp } from '@react-native-firebase/app';
import { getMessaging, subscribeToTopic, unsubscribeFromTopic } from '@react-native-firebase/messaging';

// One FCM topic per line: the backend sends to the topic and never knows devices.
export const topicFor = (code: number) => `linha-${code}`;

// A failed (un)subscription must never block favoriting: favorites are
// subscribed again on every app start.
export async function subscribeLine(code: number): Promise<void> {
  await subscribeToTopic(getMessaging(getApp()), topicFor(code)).catch(() => {});
}

export async function unsubscribeLine(code: number): Promise<void> {
  await unsubscribeFromTopic(getMessaging(getApp()), topicFor(code)).catch(() => {});
}
