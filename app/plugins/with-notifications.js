// Native side of push notifications, applied at prebuild:
// - copies the per-line small icons (ic_line_N) into the Android resources;
// - creates the single "status" channel when the app starts.
// The channel name is shown in Android's notification settings.
const fs = require('fs');
const path = require('path');
const { withDangerousMod, withMainApplication } = require('expo/config-plugins');

const ICONS = 'assets/notification';
const CHANNEL = `
    if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.O) {
      getSystemService(android.app.NotificationManager::class.java).createNotificationChannel(
        android.app.NotificationChannel("status", "Status das linhas", android.app.NotificationManager.IMPORTANCE_HIGH)
      )
    }`;

const withLineIcons = (config) =>
  withDangerousMod(config, [
    'android',
    (config) => {
      const from = path.join(config.modRequest.projectRoot, ICONS);
      const res = path.join(config.modRequest.platformProjectRoot, 'app/src/main/res');
      for (const dir of fs.readdirSync(from)) {
        fs.cpSync(path.join(from, dir), path.join(res, dir), { recursive: true });
      }
      return config;
    },
  ]);

const withStatusChannel = (config) =>
  withMainApplication(config, (config) => {
    const src = config.modResults.contents;
    if (!src.includes('NotificationChannel("status"')) {
      config.modResults.contents = src.replace(/super\.onCreate\(\)/, (m) => m + CHANNEL);
    }
    return config;
  });

module.exports = (config) => withStatusChannel(withLineIcons(config));
