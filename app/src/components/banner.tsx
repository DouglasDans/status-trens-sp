import { Pressable, StyleSheet, Text, View } from 'react-native';

import { fonts, icons, type Theme } from '@/theme/tokens';

import { StatusDot } from './status-dot';

type Props = {
  theme: Theme;
  title: string;
  body?: string;
  icon?: string;
  action?: { label: string; onPress: () => void };
};

export function Banner({ theme: t, title, body, icon = icons.info, action }: Props) {
  return (
    <View style={[styles.banner, { backgroundColor: t.banner }]}>
      <View style={styles.content}>
        <View style={{ marginTop: 1 }}>
          <StatusDot fill={t.tones.gray.fill} glyph={t.tones.gray.glyph} icon={icon} size={20} />
        </View>
        <View style={styles.texts}>
          <Text style={[styles.title, { color: t.text }]}>{title}</Text>
          {body ? <Text style={[styles.body, { color: t.muted }]}>{body}</Text> : null}
        </View>
      </View>
      {action && (
        <View style={styles.actions}>
          <Pressable
            onPress={action.onPress}
            accessibilityRole="button"
            style={({ pressed }) => [styles.action, pressed && { backgroundColor: 'rgba(127,127,127,.12)' }]}>
            <Text style={[styles.actionText, { color: t.text }]}>{action.label}</Text>
          </Pressable>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  banner: { marginHorizontal: 12, marginBottom: 8, borderRadius: 16, paddingVertical: 14, paddingHorizontal: 16, gap: 6 },
  content: { flexDirection: 'row', gap: 10, alignItems: 'flex-start' },
  texts: { flex: 1, gap: 2 },
  title: { fontFamily: fonts.semibold, fontSize: 15, lineHeight: 20 },
  body: { fontFamily: fonts.regular, fontSize: 14, lineHeight: 20 },
  actions: { flexDirection: 'row', justifyContent: 'flex-end' },
  action: { height: 40, paddingHorizontal: 14, borderRadius: 20, justifyContent: 'center' },
  actionText: { fontFamily: fonts.semibold, fontSize: 14 },
});
