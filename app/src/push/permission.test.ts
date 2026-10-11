import AsyncStorage from '@react-native-async-storage/async-storage';
import { PermissionsAndroid, Platform } from 'react-native';

import { notificationPermission, requestNotificationPermission } from './permission';

const setVersion = (v: number) => Object.defineProperty(Platform, 'Version', { get: () => v, configurable: true });
Object.defineProperty(Platform, 'OS', { get: () => 'android', configurable: true });

beforeEach(async () => {
  await AsyncStorage.clear();
  setVersion(34);
});

describe('notificationPermission', () => {
  it('is granted before Android 13, where there is nothing to ask', async () => {
    setVersion(32);
    const check = jest.spyOn(PermissionsAndroid, 'check');

    expect(await notificationPermission()).toBe('granted');
    expect(check).not.toHaveBeenCalled();
  });

  it('is granted when Android says so', async () => {
    jest.spyOn(PermissionsAndroid, 'check').mockResolvedValue(true);

    expect(await notificationPermission()).toBe('granted');
  });

  it('is undetermined when never asked', async () => {
    jest.spyOn(PermissionsAndroid, 'check').mockResolvedValue(false);

    expect(await notificationPermission()).toBe('undetermined');
  });

  it('is denied after the user refused it', async () => {
    jest.spyOn(PermissionsAndroid, 'check').mockResolvedValue(false);
    jest.spyOn(PermissionsAndroid, 'request').mockResolvedValue('denied');
    await requestNotificationPermission();

    expect(await notificationPermission()).toBe('denied');
  });

  it('is granted when enabled in the settings after a refusal', async () => {
    jest.spyOn(PermissionsAndroid, 'request').mockResolvedValue('never_ask_again');
    await requestNotificationPermission();
    jest.spyOn(PermissionsAndroid, 'check').mockResolvedValue(true);

    expect(await notificationPermission()).toBe('granted');
  });
});

describe('requestNotificationPermission', () => {
  it.each([
    ['granted', 'granted'],
    ['denied', 'denied'],
    ['never_ask_again', 'denied'],
  ] as const)('maps the Android answer %s to %s', async (answer, expected) => {
    jest.spyOn(PermissionsAndroid, 'request').mockResolvedValue(answer);

    expect(await requestNotificationPermission()).toBe(expected);
  });

  it('asks for the notification permission', async () => {
    const request = jest.spyOn(PermissionsAndroid, 'request').mockResolvedValue('granted');

    await requestNotificationPermission();

    expect(request).toHaveBeenCalledWith('android.permission.POST_NOTIFICATIONS');
  });

  it('is granted without asking before Android 13', async () => {
    setVersion(32);
    const request = jest.spyOn(PermissionsAndroid, 'request');

    expect(await requestNotificationPermission()).toBe('granted');
    expect(request).not.toHaveBeenCalled();
  });
});
