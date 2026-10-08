import AsyncStorage from '@react-native-async-storage/async-storage';

const KEY = 'favorites';

export async function loadFavorites(): Promise<number[]> {
  try {
    const raw = await AsyncStorage.getItem(KEY);
    const parsed: unknown = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed.filter((v): v is number => Number.isInteger(v)) : [];
  } catch {
    return [];
  }
}

export async function saveFavorites(favorites: number[]): Promise<void> {
  await AsyncStorage.setItem(KEY, JSON.stringify(favorites));
}
