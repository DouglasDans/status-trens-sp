import mockAsyncStorage from '@react-native-async-storage/async-storage/jest/async-storage-mock';

jest.mock('@react-native-async-storage/async-storage', () => mockAsyncStorage);

// Firebase is native: tests get inert stand-ins for the modular API.
jest.mock('@react-native-firebase/app', () => ({ getApp: jest.fn(() => ({})) }));
jest.mock('@react-native-firebase/messaging', () => ({
  getMessaging: jest.fn(() => ({})),
  subscribeToTopic: jest.fn(() => Promise.resolve()),
  unsubscribeFromTopic: jest.fn(() => Promise.resolve()),
}));
