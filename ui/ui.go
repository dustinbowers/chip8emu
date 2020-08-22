package ui

import (
	"github.com/veandco/go-sdl2/sdl"
)

var (
	width       int32
	height      int32
	rows        int32
	cols        int32
	blockWidth  int32
	blockHeight int32
)

var window *sdl.Window

func Init(screenWidth int, screenHeight int, screenCols int, screenRows int) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}

	width = int32(screenWidth)
	height = int32(screenHeight)
	cols = int32(screenCols)
	rows = int32(screenRows)
	blockWidth = width / cols
	blockHeight = height / rows

	win, err := sdl.CreateWindow("Chip8", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		width, height, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	window = win
}

func Cleanup() {
	sdl.Quit()
	window.Destroy()
}

func Draw(cells [64][32]uint8) {
	surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}
	surface.FillRect(nil, 0)

	for x, col := range cells {
		for y, cell := range col {

			xPos := int32(x) * blockWidth
			yPos := int32(y) * blockHeight

			// Yes, it is inefficient to re-draw the entire screen when not needed.
			// It's done to ensure that each frame's blitting ops take approximately
			// the same amount of time to complete regardless of 'on' pixels
			var color uint32 = 0x00000000
			if cell == 1 {
				color = 0xffffffff
			}

			rect := sdl.Rect{
				X: xPos,
				Y: yPos,
				W: blockWidth,
				H: blockHeight,
			}
			_ = surface.FillRect(&rect, color)
		}
	}
	_ = window.UpdateSurface()
}
