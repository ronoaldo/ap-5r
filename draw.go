package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"strconv"

	"github.com/golang/freetype/truetype"
	"github.com/ronoaldo/swgoh/swgohhelp"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"gopkg.in/fogleman/gg.v1"
)

var (
	fontBold    *truetype.Font
	fontRegular *truetype.Font

	gearColors = map[int]string{
		0:  "#ffffff",
		1:  "#a5d0da",
		2:  "#98fd33",
		3:  "#98fd33",
		4:  "#00bdfe",
		5:  "#00bdfe",
		6:  "#00bdfe",
		7:  "#9241ff",
		8:  "#9241ff",
		9:  "#9241ff",
		10: "#9241ff",
		11: "#9241ff",
		12: "#ffd036",
	}
)

func init() {
	var err error
	fontRegular, err = truetype.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	}
	fontBold, err = truetype.Parse(gobold.TTF)
	if err != nil {
		panic(err)
	}
}

type drawer struct {
	player swgohhelp.Player

	bold bool
	size float64

	color string

	x float64
	y float64

	ax float64
	ay float64

	advanceX float64
	advanceY float64
}

// DrawCharacterStats draws character unit stats with a beautiful image
func (d *drawer) DrawCharacterStats(u *swgohhelp.Unit) ([]byte, error) {
	// Load drawing assets
	bg, err := loadAsset("ui/ap-5r-char-stats_background.png")
	if err != nil {
		return nil, err
	}
	charOverlay, err := loadAsset("ui/ap-5r-char-stats_char-overlay.png")
	if err != nil {
		return nil, err
	}
	starYellow, err := loadAsset("ui/ap-5r-char-stats_star-yellow.png")
	if err != nil {
		return nil, err
	}
	starGray, err := loadAsset("ui/ap-5r-char-stats_star-gray.png")
	if err != nil {
		return nil, err
	}
	zeta, err := loadAsset("ui/ap-5r-char-stats_zeta.png")
	if err != nil {
		return nil, err
	}
	char, err := loadAsset(fmt.Sprintf("characters/%s.png", u.Name))
	if err != nil {
		logger.Errorf("Error loading character image %v", u.Name)
	}

	// Prepare unit canvas
	canvas := gg.NewContextForImage(bg)
	if char != nil {
		canvas.DrawImage(char, 0, 0)
	}
	canvas.DrawImage(charOverlay, 0, 0)

	// Draw char name
	d.size = 34
	d.x, d.y = 348, 70
	d.textCenter()
	d.bold = true
	d.printf(canvas, u.Name)

	// Draw char level
	d.x, d.y = 52, 720
	d.printf(canvas, "%d", u.Level)

	// Draw stars
	for i := 1; i <= 7; i++ {
		x := 55 + i*30
		y := 695
		if i > 1 {
			x += (i - 1) * 15
		}
		if i <= u.Rarity {
			canvas.DrawImage(starYellow, x, y)
		} else {
			canvas.DrawImage(starGray, x, y)
		}
	}
	// Draw gear level
	gearColor, ok := gearColors[u.Gear]
	if !ok {
		gearColor = "#ffffff"
	}
	d.x, d.y = 670, 715
	d.color = gearColor
	d.textRight()
	d.printf(canvas, "Gear Lvl %d", u.Gear)

	// Start writting stats
	d.x, d.y = 1024, 70
	d.color = "#ffffff"
	d.size, d.bold = 34, true
	d.textCenter()
	d.printf(canvas, "Character Stats")

	// Write stats labels
	d.x, d.y = 1012, 140
	d.textRight()
	d.bold = false
	d.size = 30
	for _, s := range []string{"Health", "Speed", "Potency", "Physical Damage",
		"Physical Crit. Chance", "Armor", "Physical Crit. Avoid."} {
		d.printf(canvas, s+":")
		d.y += 110
	}
	d.y = 185
	for _, s := range []string{"Protection", "Critical Damage", "Tenacity", "Special Damage",
		"Special Crit. Chance", "Resistance", "Special Crit. Avoid."} {
		d.printf(canvas, s+":")
		d.y += 110
	}

	// Write stat values
	stats, mods := u.Stats.Final, u.Stats.FromMods

	d.x, d.y = 1048, 140
	d.textLeft()
	d.bold = true

	d.printStatValue(canvas, stats.Health, mods.Health)
	d.y += 45
	d.printStatValue(canvas, stats.Protection, mods.Protection)
	d.y += 65

	d.printStatValue(canvas, stats.Speed, mods.Speed)
	d.y += 45
	d.printStatValue(canvas, stats.CriticalDamage, mods.CriticalDamage)
	d.y += 65

	d.printStatValue(canvas, stats.Potency, mods.Potency)
	d.y += 45
	d.printStatValue(canvas, stats.Tenacity, mods.Tenacity)
	d.y += 65

	d.printStatValue(canvas, stats.PhysicalDamage, mods.PhysicalDamage)
	d.y += 45
	d.printStatValue(canvas, stats.SpecialDamage, mods.SpecialDamage)
	d.y += 65

	d.printStatValue(canvas, stats.PhysicalCriticalChance, mods.PhysicalCriticalChance)
	d.y += 45
	d.printStatValue(canvas, stats.SpecialCriticalChance, mods.SpecialCriticalChance)
	d.y += 65

	d.printStatValue(canvas, stats.Armor, mods.Armor)
	d.y += 45
	d.printStatValue(canvas, stats.Resistance, mods.Resistance)
	d.y += 65

	d.printStatValue(canvas, stats.PhysicalCriticalAvoidance, mods.PhysicalCriticalAvoidance)
	d.y += 45
	d.printStatValue(canvas, stats.SpecialCriticalAvoidance, mods.SpecialCriticalAvoidance)
	d.y += 65

	// Draw zetas
	d.x, d.y = 80, 765
	d.textLeft()
	for _, skill := range u.Skills {
		if skill.IsZeta && skill.Tier == 8 {
			canvas.DrawImage(zeta, int(d.x-50), int(d.y-10))
			d.printf(canvas, skill.Name)
			d.y += 40
		}
	}

	// Draw player info at bottom
	d.x, d.y = 30, 895
	d.textLeft()
	d.printf(canvas, "%s - ", d.player.Name)
	d.x += d.advanceX
	d.color = "#00bdfe"
	d.printf(canvas, "%s", d.player.Titles.Selected)

	var b bytes.Buffer
	if err := canvas.EncodePNG(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// DrawUnitList draw a unit list, 5 per row.
// Usefull for drawing a team or a roster of units by category
func (d *drawer) DrawUnitList(units []swgohhelp.Unit) ([]byte, error) {
	// 5 per row, 100px per unit, 10px padding
	padding := 30
	portraitSize := 100
	unitSize := portraitSize + padding*2
	width := unitSize * 5
	height := unitSize * int(math.Ceil((float64(len(units)) / 5.0)))

	canvas := gg.NewContext(width, height)
	canvas.SetHexColor("#0D1D25")
	canvas.Clear()

	// Use an asset bundle to save some disk I/O
	bundle := &assetBundle{
		ui: make(map[string]image.Image),
	}

	// draw each unit portrait
	x, y := padding, padding
	for unitCount, u := range units {
		// Draw portrait
		portrait, err := loadAsset(fmt.Sprintf("characters/%s_portrait.png", u.Name))
		if err != nil {
			logger.Errorf("Error loading character image portrait %v: %v", u.Name, err)
			continue
		}
		croppedPortrait := cropCircle(portrait)
		canvas.DrawImage(croppedPortrait, x, y)

		// Draw gear
		gear, _ := bundle.loadUIAsset(fmt.Sprintf("ui/gear-icon-g%d_100x100.png", u.Gear))
		if gear != nil {
			canvas.DrawImage(gear, x, y)
		}

		// Draw stars
		starYellow, _ := bundle.loadUIAsset("ui/ap-5r-char-portrait_star-yellow.png")
		starGray, _ := bundle.loadUIAsset("ui/ap-5r-char-portrait_star-gray.png")
		if starYellow != nil {
			cx, cy := x+(unitSize/4)+10, y+(unitSize/4)
			rotate := []float64{0, -66, -43, -21, 0, 21, 43, 66}
			for i := 1; i <= 7; i++ {
				canvas.Push()
				canvas.Stroke()
				canvas.Translate(0.5, 0)
				canvas.RotateAbout(gg.Radians(rotate[i]), f(cx), f(cy))
				if u.Rarity >= i {
					canvas.DrawImageAnchored(starYellow, cx, cy-26, 0.5, 0.5)
				} else {
					canvas.DrawImageAnchored(starGray, cx, cy-26, 0.5, 0.5)
				}
				canvas.Pop()
			}
		}

		// Check offset
		x += unitSize
		if (unitCount+1)%5 == 0 {
			y += unitSize
			x = padding
		}
	}

	var b bytes.Buffer
	if err := canvas.EncodePNG(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (d *drawer) printStatValue(canvas *gg.Context, v interface{}, m interface{}) {
	switch v.(type) {
	case int:
		d.printf(canvas, "%d", v)
		d.x += d.advanceX
		if m.(int) > 0 {
			d.color = "#00bdfe"
			d.printf(canvas, "(%d)", m)
		}
	case float64:
		d.printf(canvas, "%.01f%%", v.(float64)*100)
		d.x += d.advanceX
		if m.(float64) > 0 {
			d.color = "#00bdfe"
			d.printf(canvas, "(%.01f%%)", m.(float64)*100)
		}
	}
	d.x = 1048
	d.color = "#ffffff"
}

func (d *drawer) printf(canvas *gg.Context, format string, args ...interface{}) error {
	text := fmt.Sprintf(format, args...)
	// Draw background text dark bold
	fontFace, err := loadFont(d.size, d.bold)
	if err != nil {
		return err
	}
	canvas.SetFontFace(fontFace)
	canvas.SetRGB(0, 0, 0)
	canvas.DrawStringAnchored(text, d.x, d.y, d.ax, d.ay)

	// Draw background text dark bold
	fontFace, err = loadFont(d.size, d.bold)
	if err != nil {
		return err
	}
	canvas.SetFontFace(fontFace)
	if d.color == "" {
		d.color = "#ffffff"
	}
	canvas.SetHexColor(d.color)
	canvas.DrawStringAnchored(text, d.x-2, d.y-2, d.ax, d.ay)
	d.advanceX, d.advanceY = canvas.MeasureString(text + " ")
	return nil
}

func (d *drawer) textCenter() {
	d.ax, d.ay = 0.5, 0.5
}

func (d *drawer) textLeft() {
	d.ax, d.ay = 0, 0.5
}

func (d *drawer) textRight() {
	d.ax, d.ay = 1, 0.5
}

type assetBundle struct {
	ui map[string]image.Image
}

func (a *assetBundle) loadUIAsset(file string) (image.Image, error) {
	if img, ok := a.ui[file]; ok {
		return img, nil
	}

	img, err := loadAsset(file)
	if err != nil {
		return nil, err
	}

	a.ui[file] = img
	return img, nil
}

func loadAsset(file string) (image.Image, error) {
	assetDir := os.Getenv("BOT_ASSET_DIR")
	if assetDir == "" {
		assetDir = "."
	}
	return gg.LoadPNG(assetDir + "/images/" + file)
}

func loadFont(size float64, bold bool) (font.Face, error) {
	font := fontRegular
	if bold {
		font = fontBold
	}
	face := truetype.NewFace(font, &truetype.Options{Size: size})
	return face, nil
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func ftoa(v float64) string {
	return fmt.Sprintf("%.02f%%", v*100)
}

type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

func cropCircle(src image.Image) image.Image {
	dst := image.NewRGBA(src.Bounds())
	p := image.Point{X: 50, Y: 50}
	r := 50
	draw.DrawMask(dst, dst.Bounds(), src, image.ZP, &circle{p, r}, image.ZP, draw.Over)
	return dst
}

func f(i int) float64 {
	return float64(i)
}
