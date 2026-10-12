// Command badges renders the colored line badges sent as the notification
// picture (linha-<code>.png), one per catalog line, with ImageMagick 7.
// The output is published to the folder NOTIFICATION_IMAGE_BASE_URL points at;
// rerun it when a line is added to the catalog or changes color.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/DouglasDans/status-trens-sp/backend/internal/catalog"
)

func main() {
	out := flag.String("out", "badges", "output folder")
	font := flag.String("font", "/usr/share/fonts/open-sans/OpenSans-Bold.ttf", "bold font for the line number")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	// Same proportions as the app's line badge: radius a quarter of the side,
	// two digit numbers a bit smaller.
	const size, radius = 192, 48
	for _, l := range catalog.Lines {
		num := strconv.Itoa(l.Code)
		pt := 104
		if len(num) > 1 {
			pt = 88
		}
		path := filepath.Join(*out, fmt.Sprintf("linha-%d.png", l.Code))
		cmd := exec.Command("magick",
			"-size", fmt.Sprintf("%dx%d", size, size), "xc:none",
			"-fill", l.ColorHex, "-draw", fmt.Sprintf("roundrectangle 0,0 %d,%d %d,%d", size-1, size-1, radius, radius),
			"(", "-background", "none", "-fill", l.TextColorHex, "-font", *font, "-pointsize", strconv.Itoa(pt), "-density", "72", "label:"+num, "-trim", "+repage", ")",
			"-gravity", "center", "-composite", "PNG32:"+path)
		if b, err := cmd.CombinedOutput(); err != nil {
			log.Fatalf("line %d: %v: %s", l.Code, err, b)
		}
		fmt.Println(path)
	}
}
