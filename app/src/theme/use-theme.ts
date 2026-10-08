import { useColorScheme } from 'react-native';

import { themes } from './tokens';

export function useTheme() {
  return useColorScheme() === 'dark' ? themes.dark : themes.light;
}
