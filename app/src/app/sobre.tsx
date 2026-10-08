import { router } from 'expo-router';
import { Linking, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import Svg, { Path } from 'react-native-svg';

import { fonts, type Theme } from '@/theme/tokens';
import { useTheme } from '@/theme/use-theme';

// Required by Google Play for apps showing government-sourced information:
// clear non-affiliation notice and visible data sources.
export default function Sobre() {
  const t = useTheme();
  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: t.bg }]}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Voltar"
          style={styles.back}>
          <Svg width={22} height={22} viewBox="0 0 24 24">
            <Path d="M15 5l-7 7 7 7" fill="none" stroke={t.text} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />
          </Svg>
        </Pressable>
        <Text style={[styles.title, { color: t.text }]}>Sobre</Text>
      </View>
      <ScrollView contentContainerStyle={styles.content}>
        <Section theme={t} title="APP NÃO OFICIAL">
          <Paragraph theme={t}>
            O Status Trens SP não é afiliado ao Metrô, CPTM, ViaQuatro, ViaMobilidade, TIC Trens, Linha Uni, Trivia
            Trens nem ao Governo do Estado de São Paulo.
          </Paragraph>
        </Section>
        <Section theme={t} title="FONTES DOS DADOS">
          <Paragraph theme={t}>
            O status das linhas vem do serviço Próximo Trem e, como alternativa, do serviço público da CPTM. Pode haver
            atraso em relação aos canais oficiais.
          </Paragraph>
          <Paragraph theme={t}>Informações oficiais:</Paragraph>
          <LinkText theme={t} url="https://www.metro.sp.gov.br" />
          <LinkText theme={t} url="https://www.cptm.sp.gov.br" />
        </Section>
        <Section theme={t} title="PRIVACIDADE">
          <Paragraph theme={t}>
            O app não tem login e não coleta dados pessoais. Suas linhas favoritas ficam salvas apenas neste aparelho.
          </Paragraph>
        </Section>
      </ScrollView>
    </SafeAreaView>
  );
}

function Section({ theme: t, title, children }: { theme: Theme; title: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: t.muted }]}>{title}</Text>
      <View style={[styles.card, { backgroundColor: t.surface }]}>{children}</View>
    </View>
  );
}

function Paragraph({ theme: t, children }: { theme: Theme; children: React.ReactNode }) {
  return <Text style={[styles.paragraph, { color: t.text }]}>{children}</Text>;
}

function LinkText({ theme: t, url }: { theme: Theme; url: string }) {
  return (
    <Text accessibilityRole="link" onPress={() => Linking.openURL(url)} style={[styles.link, { color: t.text }]}>
      {url.replace('https://', '')}
    </Text>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  header: { flexDirection: 'row', alignItems: 'center', paddingTop: 6, paddingHorizontal: 8, paddingBottom: 8 },
  back: { width: 48, height: 48, borderRadius: 24, alignItems: 'center', justifyContent: 'center' },
  title: { fontFamily: fonts.bold, fontSize: 22 },
  content: { paddingBottom: 28 },
  section: { paddingBottom: 12 },
  sectionTitle: { fontFamily: fonts.monoSemibold, fontSize: 11, letterSpacing: 0.88, paddingHorizontal: 20, paddingTop: 10, paddingBottom: 8 },
  card: { marginHorizontal: 12, borderRadius: 16, padding: 16, gap: 10 },
  paragraph: { fontFamily: fonts.regular, fontSize: 15, lineHeight: 22 },
  link: { fontFamily: fonts.medium, fontSize: 15, textDecorationLine: 'underline' },
});
