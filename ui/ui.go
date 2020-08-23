package ui

// typedef unsigned char Uint8;
// void SineWave(void *userdata, Uint8 *stream, int len);
import "C"
import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"math"
	"reflect"
	"unsafe"
)

var (
	width       int32
	height      int32
	rows        int32
	cols        int32
	blockWidth  int32
	blockHeight int32
)

const (
	DefaultFrequency = 16000
	DefaultFormat    = sdl.AUDIO_S16
	DefaultChannels  = 2
	DefaultSamples   = 512

	toneHz = 200
	dPhase = 2 * math.Pi * toneHz / DefaultSamples
)

var window *sdl.Window
var audioDev sdl.AudioDeviceID

func Init(screenWidth int, screenHeight int, screenCols int, screenRows int) {
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
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

	// Audio
	// Specify the configuration for our default playback device
	spec := sdl.AudioSpec{
		Freq:     DefaultFrequency,
		Format:   DefaultFormat,
		Channels: DefaultChannels,
		Samples:  DefaultSamples,
		Callback: sdl.AudioCallback(C.SineWave),
	}

	// Open default playback device
	if audioDev, err = sdl.OpenAudioDevice("", false, &spec, nil, 0); err != nil {
		log.Println(err)
		return
	}
}

func Draw(cells [64][32]uint8) error {
	surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}
	err = surface.FillRect(nil, 0)
	if err != nil {
		return fmt.Errorf("draw: FillRect failed: %v", err)
	}

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
	err = window.UpdateSurface()
	if err != nil {
		return fmt.Errorf("draw: UpdateSurface failed: %v", err)
	}
	return nil
}

func Beep(on bool) {
	sdl.PauseAudioDevice(audioDev, !on)
	//go func() {
	//	sdl.PauseAudioDevice(audioDev, false)
	//	time.Sleep(time.Millisecond * 50)
	//	sdl.PauseAudioDevice(audioDev, true)
	//}()
}

//export SineWave
func SineWave(userdata unsafe.Pointer, stream *C.Uint8, length C.int) {
	n := int(length) / 2
	hdr := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(stream)), Len: n, Cap: n}
	buf := *(*[]C.ushort)(unsafe.Pointer(&hdr))

	var phase float64
	for i := 0; i < n; i++ {
		phase += dPhase
		sample := C.ushort((math.Sin(phase) + 0.999999) * 32768)
		buf[i] = sample
	}
}

func Cleanup() {
	sdl.Quit()
	sdl.CloseAudioDevice(audioDev)
	_ = window.Destroy()
}
