import { Modal, Pressable, StyleSheet, Text, View } from 'react-native';
import Svg, { Path } from 'react-native-svg';

import { bell, fonts, type Theme } from '@/theme/tokens';

type Props = { visible: boolean; theme: Theme; onLater: () => void; onContinue: () => void };

// Explains the notifications before Android's own permission dialog.
export function PermissionSheet({ visible, theme: t, onLater, onContinue }: Props) {
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onLater}>
      <Pressable style={[styles.scrim, { backgroundColor: t.scrim }]} onPress={onLater} accessibilityLabel="Fechar" />
      <View style={[styles.sheet, { backgroundColor: t.sheet }]}>
        <View style={[styles.handle, { backgroundColor: t.dash }]} />
        <View style={[styles.icon, { backgroundColor: t.banner }]}>
          <Svg width={24} height={24} viewBox="0 0 24 24">
            <Path d={bell} fill="none" stroke={t.text} strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" />
          </Svg>
        </View>
        <Text style={[styles.title, { color: t.text }]}>Avisar quando suas linhas mudarem?</Text>
        <View style={styles.texts}>
          <Text style={[styles.text, { color: t.muted }]}>
            Quando uma linha favoritada mudar de status, você recebe uma notificação.
          </Text>
          <Text style={[styles.text, { color: t.muted }]}>Abertura e encerramento normais do dia não geram aviso.</Text>
          <Text style={[styles.text, { color: t.muted }]}>Em seguida, o Android pede a permissão.</Text>
        </View>
        <View style={styles.actions}>
          <Pressable
            onPress={onLater}
            accessibilityRole="button"
            style={({ pressed }) => [styles.button, pressed && { backgroundColor: 'rgba(127,127,127,.12)' }]}>
            <Text style={[styles.buttonText, { color: t.text }]}>Agora não</Text>
          </Pressable>
          <Pressable
            onPress={onContinue}
            accessibilityRole="button"
            style={[styles.button, styles.primary, { backgroundColor: t.primary }]}>
            <Text style={[styles.buttonText, { color: t.primaryInk }]}>Continuar</Text>
          </Pressable>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  scrim: { flex: 1 },
  sheet: { borderTopLeftRadius: 28, borderTopRightRadius: 28, paddingTop: 12, paddingHorizontal: 24, paddingBottom: 40, gap: 16 },
  handle: { width: 32, height: 4, borderRadius: 2, alignSelf: 'center' },
  icon: { width: 48, height: 48, borderRadius: 24, alignItems: 'center', justifyContent: 'center' },
  title: { fontFamily: fonts.bold, fontSize: 22, lineHeight: 27.5, letterSpacing: -0.22 },
  texts: { gap: 10 },
  text: { fontFamily: fonts.regular, fontSize: 15, lineHeight: 22.5 },
  actions: { flexDirection: 'row', justifyContent: 'flex-end', gap: 8, marginTop: 4 },
  button: { height: 48, paddingHorizontal: 20, borderRadius: 24, justifyContent: 'center' },
  primary: { paddingHorizontal: 24 },
  buttonText: { fontFamily: fonts.semibold, fontSize: 15 },
});
