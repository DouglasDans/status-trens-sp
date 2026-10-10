// SVGs are bundled as plain assets (rendered by expo-image), not components.
declare module '*.svg' {
  const asset: number;
  export default asset;
}
