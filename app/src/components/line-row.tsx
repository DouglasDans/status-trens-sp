import { Image } from 'expo-image';
import { Pressable, StyleSheet, Text, useColorScheme, View } from 'react-native';
import Svg, { Path } from 'react-native-svg';

import type { LineStatus } from '@/api/status';
import { SHOW_OPERATOR_LOGO } from '@/config';
import { logoFor } from '@/operators';
import { badgeInk, lineInk } from '@/theme/contrast';
import { fonts, star, stateVisual, type Theme } from '@/theme/tokens';

import { StatusDot } from './status-dot';

type Props = {
  line: LineStatus;
  theme: Theme;
  favorite: boolean;
  grayBadge: boolean; // late night: every line closed, badges go gray
  first: boolean;
  onToggleFavorite: () => void;
};

export function LineRow({ line, theme: t, favorite, grayBadge, first, onToggleFavorite }: Props) {
  const visual = stateVisual[line.status] ?? stateVisual.desconhecido;
  const tone = t.tones[visual.tone];
  const badge = grayBadge ? t.grayBadge : line.colorHex;
  const ink = grayBadge ? badgeInk(badge) : lineInk(badge, line.textColorHex);
  const num = String(line.code);
  const a11yName = `linha ${line.code} ${line.name}`;

  return (
    <View style={[styles.row, !first && { borderTopWidth: 1, borderTopColor: t.border }]}>
      <View style={[styles.badge, { backgroundColor: badge }]}>
        <Text style={{ color: ink, fontFamily: fonts.badge, fontSize: num.length > 1 ? 15 : 17 }}>
          {num}
        </Text>
      </View>

      <View style={styles.body}>
        <Text style={[styles.name, { color: t.text }]}>{line.name}</Text>
        <View style={styles.statusRow}>
          <StatusDot fill={tone.fill} glyph={tone.glyph} icon={visual.icon} />
          <Text style={[styles.label, { color: tone.label }]}>{line.label || visual.label}</Text>
        </View>
        {line.description !== '' && (
          <Text style={[styles.description, { color: t.muted }]}>{line.description}</Text>
        )}
      </View>

      {/* Operator and star are centered on the whole row, side by side. */}
      {SHOW_OPERATOR_LOGO && line.operator !== '' && <OperatorBadge line={line} theme={t} />}

      <Pressable
        onPress={onToggleFavorite}
        accessibilityRole="button"
        accessibilityLabel={favorite ? `Remover ${a11yName} dos favoritos` : `Favoritar ${a11yName}`}
        style={({ pressed }) => [styles.star, pressed && { backgroundColor: 'rgba(127,127,127,.12)' }]}>
        <Svg width={22} height={22} viewBox="0 0 24 24">
          <Path
            d={star}
            fill={favorite ? t.text : 'none'}
            stroke={favorite ? t.text : t.muted}
            strokeWidth={1.8}
            strokeLinejoin="round"
          />
        </Svg>
      </Pressable>
    </View>
  );
}

function OperatorBadge({ line, theme: t }: { line: LineStatus; theme: Theme }) {
  const logo = logoFor(line.operator, useColorScheme() === 'dark' ? 'dark' : 'light');
  const label = `Operadora ${line.operatorName || line.operator}`;
  if (logo) {
    return (
      <View accessibilityLabel={label} style={styles.logoBox}>
        <Image source={logo.source} contentFit="contain" tintColor={logo.tint} style={styles.logo} />
      </View>
    );
  }
  return (
    <View accessibilityLabel={label} style={[styles.operator, { borderColor: t.dash }]}>
      <Text style={[styles.operatorText, { color: t.muted }]}>{line.operatorName || line.operator}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', gap: 12, alignItems: 'flex-start', paddingVertical: 12, paddingLeft: 16, paddingRight: 4 },
  badge: { width: 40, height: 40, borderRadius: 10, alignItems: 'center', justifyContent: 'center', marginTop: 2 },
  body: { flex: 1, minWidth: 0, gap: 4, paddingTop: 2 },
  name: { fontFamily: fonts.semibold, fontSize: 16, lineHeight: 20 },
  operator: { alignSelf: 'center', height: 20, paddingHorizontal: 6, borderWidth: 1, borderStyle: 'dashed', borderRadius: 5, justifyContent: 'center' },
  operatorText: { fontFamily: fonts.mono, fontSize: 10 },
  logoBox: { alignSelf: 'center', height: 20, justifyContent: 'center' },
  logo: { width: 56, height: 12 },
  statusRow: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  label: { fontFamily: fonts.medium, fontSize: 14 },
  description: { fontFamily: fonts.regular, fontSize: 13, lineHeight: 19 },
  star: { alignSelf: 'center', width: 48, height: 48, borderRadius: 24, alignItems: 'center', justifyContent: 'center' },
});
