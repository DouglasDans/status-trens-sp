import { View } from 'react-native';

import type { Theme } from '@/theme/tokens';

const widths = [
  ['46%', '30%'],
  ['38%', '36%'],
  ['54%', '26%'],
  ['42%', '34%'],
  ['50%', '28%'],
] as const;

/** Placeholder list shown on the first load. */
export function Skeleton({ theme: t }: { theme: Theme }) {
  return (
    <View accessibilityLabel="Carregando linhas">
      <View style={{ paddingHorizontal: 20, paddingVertical: 8 }}>
        <View style={{ width: 80, height: 10, borderRadius: 5, backgroundColor: t.skel }} />
      </View>
      <View style={{ marginHorizontal: 12, backgroundColor: t.surface, borderRadius: 16, overflow: 'hidden' }}>
        {Array.from({ length: 9 }, (_, i) => {
          const [w1, w2] = widths[i % widths.length];
          return (
            <View
              key={i}
              style={{
                flexDirection: 'row',
                gap: 12,
                alignItems: 'center',
                paddingVertical: 14,
                paddingHorizontal: 16,
                borderTopWidth: i ? 1 : 0,
                borderTopColor: t.border,
              }}>
              <View style={{ width: 40, height: 40, borderRadius: 10, backgroundColor: t.skel }} />
              <View style={{ flex: 1, gap: 8 }}>
                <View style={{ width: w1, height: 12, borderRadius: 6, backgroundColor: t.skel }} />
                <View style={{ width: w2, height: 10, borderRadius: 5, backgroundColor: t.skel }} />
              </View>
            </View>
          );
        })}
      </View>
    </View>
  );
}
