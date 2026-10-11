import { Link } from 'expo-router';
import { useCallback, useEffect, useReducer, useRef, useState } from 'react';
import { AppState, LayoutAnimation, Pressable, RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import Svg, { Path } from 'react-native-svg';

import { fetchStatus, type LineStatus, type StatusResponse } from '@/api/status';
import { Banner } from '@/components/banner';
import { LineRow } from '@/components/line-row';
import { PermissionSheet } from '@/components/permission-sheet';
import { Skeleton } from '@/components/skeleton';
import { API_URL } from '@/config';
import { formatClock, groupLines, isNight } from '@/domain/lines';
import { initialStatusState, statusReducer } from '@/domain/status-state';
import { loadFavorites, saveFavorites } from '@/favorites/storage';
import { firebasePush, type PermissionState, type Push } from '@/push';
import { fonts, icons, star, type Theme } from '@/theme/tokens';
import { useTheme } from '@/theme/use-theme';

type Props = { load?: () => Promise<StatusResponse>; push?: Push };

export function StatusScreen({ load = () => fetchStatus(API_URL), push = firebasePush }: Props) {
  const t = useTheme();
  const [state, dispatch] = useReducer(statusReducer, initialStatusState);
  const [favorites, setFavorites] = useState<number[]>([]);
  const [permission, setPermission] = useState<PermissionState>('granted');
  const [explaining, setExplaining] = useState(false);
  const scrollRef = useRef<ScrollView>(null);

  const refresh = useCallback(async () => {
    dispatch({ type: 'refresh' });
    try {
      dispatch({ type: 'success', data: await load() });
    } catch {
      dispatch({ type: 'failure' });
    }
  }, [load]);

  useEffect(() => {
    refresh();
    loadFavorites().then((stored) => {
      setFavorites(stored);
      // Subscribing again is harmless and repairs any subscription that failed.
      stored.forEach((code) => push.subscribe(code).catch(() => {}));
    });
    const checkPermission = () => push.permission().then(setPermission);
    checkPermission();
    // The user may have changed the permission in the settings meanwhile.
    const sub = AppState.addEventListener('change', (s) => s === 'active' && checkPermission());
    return () => sub.remove();
    // Load once on mount; later loads are user-driven.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const toggleFavorite = (code: number) => {
    const adding = !favorites.includes(code);
    const next = adding ? [...favorites, code] : favorites.filter((c) => c !== code);
    // Animate the line moving between sections instead of jumping.
    LayoutAnimation.configureNext(LayoutAnimation.Presets.easeInEaseOut);
    setFavorites(next);
    saveFavorites(next);
    (adding ? push.subscribe(code) : push.unsubscribe(code)).catch(() => {});
    if (adding && permission === 'undetermined') setExplaining(true);
    // A favorited line moves to the top section; follow it so it does not
    // vanish from view when it was favorited far down the list.
    if (adding) scrollRef.current?.scrollTo({ y: 0, animated: true });
  };

  const askPermission = async () => {
    setExplaining(false);
    setPermission(await push.requestPermission());
  };

  const { data } = state;
  const clock = data ? formatClock(new Date(data.updatedAt)) : '';
  const night = data ? isNight(data.lines) : false;
  const updated = state.refreshing && data ? 'Atualizando…' : data ? `Atualizado às ${clock}` : 'Carregando…';

  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: t.bg }]}>
      <View style={styles.header}>
        <View style={styles.headerTexts}>
          <Text style={[styles.title, { color: t.text }]}>Status Trens SP</Text>
          <Pressable onPress={refresh} disabled={state.refreshing} accessibilityRole="button">
            <Text style={[styles.updated, { color: t.muted }]}>{updated}</Text>
          </Pressable>
        </View>
        <Link href="/sobre" asChild>
          <Pressable accessibilityRole="link" accessibilityLabel="Sobre o app" style={styles.about}>
            <Svg width={22} height={22} viewBox="0 0 16 16">
              <Path d="M8 1.8a6.2 6.2 0 1 0 0 12.4A6.2 6.2 0 0 0 8 1.8z" fill="none" stroke={t.muted} strokeWidth={1.4} />
              <Path d={icons.info} fill="none" stroke={t.muted} strokeWidth={1.6} strokeLinecap="round" />
            </Svg>
          </Pressable>
        </Link>
      </View>

      {state.error && data && (
        <Banner
          theme={t}
          title="Não foi possível atualizar"
          body={`Mostrando os dados das ${clock}.`}
          action={{ label: 'Tentar de novo', onPress: refresh }}
        />
      )}
      {state.error && !data && (
        <Banner
          theme={t}
          title="Não foi possível carregar"
          body="Confira sua conexão e tente de novo."
          action={{ label: 'Tentar de novo', onPress: refresh }}
        />
      )}
      {permission === 'denied' && favorites.length > 0 && (
        <Banner
          theme={t}
          title="Notificações desativadas"
          body="Seus favoritos continuam funcionando, mas sem aviso quando o status mudar."
          icon={icons.bang}
          action={{ label: 'Ativar nas configurações', onPress: push.openSettings }}
        />
      )}
      {night && <Banner theme={t} title="Todas as linhas estão com operação encerrada" icon={icons.moon} />}

      <ScrollView
        ref={scrollRef}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl refreshing={state.refreshing && !!data} onRefresh={refresh} tintColor={t.text} />
        }>
        {state.loading && !data && <Skeleton theme={t} />}
        {data && (
          <Groups
            lines={data.lines}
            favorites={favorites}
            night={night}
            theme={t}
            onToggleFavorite={toggleFavorite}
          />
        )}
      </ScrollView>

      <PermissionSheet visible={explaining} theme={t} onLater={() => setExplaining(false)} onContinue={askPermission} />
    </SafeAreaView>
  );
}

type GroupsProps = {
  lines: LineStatus[];
  favorites: number[];
  night: boolean;
  theme: Theme;
  onToggleFavorite: (code: number) => void;
};

function Groups({ lines, favorites: favoriteCodes, night, theme: t, onToggleFavorite }: GroupsProps) {
  const { favorites, others } = groupLines(lines, favoriteCodes);

  const card = (group: LineStatus[]) => (
    <View style={[styles.card, { backgroundColor: t.surface }]}>
      {group.map((line, i) => (
        <LineRow
          key={line.code}
          line={line}
          theme={t}
          favorite={favoriteCodes.includes(line.code)}
          grayBadge={night}
          first={i === 0}
          onToggleFavorite={() => onToggleFavorite(line.code)}
        />
      ))}
    </View>
  );

  return (
    <>
      <View style={styles.group}>
        <GroupHeader theme={t} title="FAVORITAS" count={favorites.length ? String(favorites.length) : ''} />
        {favorites.length ? (
          card(favorites)
        ) : (
          <View style={[styles.hint, { borderColor: t.dash }]}>
            <Svg width={20} height={20} viewBox="0 0 24 24">
              <Path d={star} fill="none" stroke={t.muted} strokeWidth={1.8} strokeLinejoin="round" />
            </Svg>
            <Text style={[styles.hintText, { color: t.muted }]}>
              Toque na estrela de uma linha para ela aparecer aqui. Você recebe um aviso quando o status dela mudar.
            </Text>
          </View>
        )}
      </View>
      <View style={styles.group}>
        <GroupHeader theme={t} title={favorites.length ? 'OUTRAS LINHAS' : 'TODAS AS LINHAS'} count={String(others.length)} />
        {card(others)}
      </View>
    </>
  );
}

function GroupHeader({ theme: t, title, count }: { theme: Theme; title: string; count: string }) {
  return (
    <View style={styles.groupHeader}>
      <Text style={[styles.groupTitle, { color: t.muted }]}>{title}</Text>
      <Text style={[styles.groupTitle, { color: t.muted }]}>{count}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  header: { flexDirection: 'row', alignItems: 'flex-start', paddingTop: 14, paddingBottom: 12, paddingLeft: 20, paddingRight: 8 },
  headerTexts: { flex: 1, gap: 4 },
  title: { fontFamily: fonts.bold, fontSize: 24, lineHeight: 29, letterSpacing: -0.24 },
  updated: { fontFamily: fonts.regular, fontSize: 13 },
  about: { width: 48, height: 48, borderRadius: 24, alignItems: 'center', justifyContent: 'center' },
  list: { paddingBottom: 28 },
  group: { paddingBottom: 12 },
  groupHeader: { flexDirection: 'row', justifyContent: 'space-between', paddingTop: 10, paddingBottom: 8, paddingHorizontal: 20 },
  groupTitle: { fontFamily: fonts.monoSemibold, fontSize: 11, letterSpacing: 0.88 },
  card: { marginHorizontal: 12, borderRadius: 16, overflow: 'hidden' },
  hint: {
    marginHorizontal: 12,
    borderWidth: 1.5,
    borderStyle: 'dashed',
    borderRadius: 16,
    paddingVertical: 14,
    paddingHorizontal: 16,
    flexDirection: 'row',
    gap: 12,
    alignItems: 'flex-start',
  },
  hintText: { flex: 1, fontFamily: fonts.regular, fontSize: 14, lineHeight: 20 },
});
