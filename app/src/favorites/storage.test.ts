import AsyncStorage from '@react-native-async-storage/async-storage';

import { loadFavorites, saveFavorites } from './storage';

beforeEach(() => AsyncStorage.clear());

describe('favorites storage', () => {
  it('returns no favorites on first launch', async () => {
    expect(await loadFavorites()).toEqual([]);
  });

  it('persists favorites between launches', async () => {
    await saveFavorites([9, 7]);

    expect(await loadFavorites()).toEqual([9, 7]);
  });

  it('recovers from corrupted data instead of crashing', async () => {
    await AsyncStorage.setItem('favorites', '{not json');

    expect(await loadFavorites()).toEqual([]);
  });

  it('drops values that are not line codes', async () => {
    await AsyncStorage.setItem('favorites', JSON.stringify([9, '7', null, 3]));

    expect(await loadFavorites()).toEqual([9, 3]);
  });
});
