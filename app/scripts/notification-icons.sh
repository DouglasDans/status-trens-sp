#!/usr/bin/env bash
# Generates the per-line notification small icons (ic_line_N): a white rounded
# square with the line number knocked out. Android only reads the alpha channel
# and tints the shape with the color sent in the push.
# Requires ImageMagick 7. Output is committed; rerun only to change the design.
set -euo pipefail

cd "$(dirname "$0")/.."
font=node_modules/@expo-google-fonts/ibm-plex-sans/700Bold/IBMPlexSans_700Bold.ttf
out=assets/notification
lines=$(seq 1 20) # numbers only, so spare codes cover lines not in the catalog yet

# density bucket and scale over the 24dp baseline
for bucket in mdpi:1 hdpi:1.5 xhdpi:2 xxhdpi:3 xxxhdpi:4; do
  name=${bucket%%:*} s=${bucket##*:}
  mkdir -p "$out/drawable-$name"
  size=$(awk "BEGIN{print int(24*$s+.5)}")
  pad=$(awk "BEGIN{print int(2*$s+.5)}")
  radius=$(awk "BEGIN{print 5*$s}")
  for n in $lines; do
    pt=$(awk "BEGIN{print (${#n}>1 ? 11 : 13.5)*$s}")
    magick -size "${size}x${size}" xc:none -fill white \
      -draw "roundrectangle $pad,$pad $((size-pad-1)),$((size-pad-1)) $radius,$radius" \
      \( -background none -fill white -font "$font" -pointsize "$pt" -density 72 label:"$n" -trim +repage \) \
      -gravity center -compose DstOut -composite \
      "PNG32:$out/drawable-$name/ic_line_$n.png"
  done
done
