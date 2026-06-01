package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// iconPNG рисует простую иконку приложения заданного размера:
// фирменный синий фон и белая стрелка «вниз» (приём файлов).
// Используется как значок на экране «Домой» iPhone.
func iconPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	bg := color.RGBA{0x4c, 0x8d, 0xff, 0xff} // тот же синий, что и на странице
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	s := float64(size)
	cx := s / 2

	// Геометрия стрелки (доли от размера иконки).
	stemHalf := s * 0.09  // полуширина «ножки»
	stemTop := s * 0.22   // верх ножки
	stemBot := s * 0.52   // низ ножки / верх головки
	headHalf := s * 0.22  // полуширина головки у основания
	headBot := s * 0.78   // остриё (низ)

	for y := 0; y < size; y++ {
		fy := float64(y)
		for x := 0; x < size; x++ {
			fx := float64(x)
			inStem := fy >= stemTop && fy <= stemBot &&
				fx >= cx-stemHalf && fx <= cx+stemHalf
			inHead := false
			if fy >= stemBot && fy <= headBot {
				// Треугольник: ширина линейно убывает к острию.
				w := headHalf * (headBot - fy) / (headBot - stemBot)
				inHead = fx >= cx-w && fx <= cx+w
			}
			if inStem || inHead {
				img.Set(x, y, white)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// webManifest — манифест PWA, чтобы значок на «Домой» вёл себя как приложение.
const webManifest = `{
  "name": "IpDrop",
  "short_name": "IpDrop",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#0b0f17",
  "theme_color": "#0b0f17",
  "icons": [
    { "src": "/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icon-512.png", "sizes": "512x512", "type": "image/png" }
  ]
}`
