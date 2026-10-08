import { View } from 'react-native';
import Svg, { Path } from 'react-native-svg';

type Props = { fill: string; glyph: string; icon: string; size?: number };

/** Round status marker with a glyph, so the state is readable without color. */
export function StatusDot({ fill, glyph, icon, size = 18 }: Props) {
  return (
    <View
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        backgroundColor: fill,
        alignItems: 'center',
        justifyContent: 'center',
      }}>
      <Svg width={size * 0.67} height={size * 0.67} viewBox="0 0 16 16">
        <Path d={icon} fill="none" stroke={glyph} strokeWidth={2.2} strokeLinecap="round" strokeLinejoin="round" />
      </Svg>
    </View>
  );
}
