package main

import (
	"chip8emu/chip8"
	"chip8emu/ui"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"os"
	"time"
)

const (
	screenCols = 64
	screenRows = 32
)

var keyMap map[int]uint8

func main() {
	var romPath string
	romPath = "roms/games/Space Invaders [David Winter].ch8"
	if len(os.Args) == 2 {
		romPath = os.Args[1]
	}

	var emu chip8.Chip8

	log.Print("Initializing emulator... ")
	emu.Initialize()
	log.Println("Done")

	log.Printf("Loading rom at: %v\n", romPath)
	err := emu.LoadRom(romPath)
	if err != nil {
		log.Printf("Rom load failed: %v", err)
		os.Exit(1)
		return
	}

	keyMap = getKeyMap()

	ui.Init(512, 256, screenCols, screenRows)
	defer ui.Cleanup()

	running := true
	hz := 700
	delay := time.Duration(1000 / hz)
	go func() {
		log.Println("Starting... ")
		for {
			_, err = emu.EmulateCycle()
			if err != nil {
				log.Fatalf("emu.EmulateCycle: %v", err)
			}
			if running == false {
				return
			}
			time.Sleep(time.Microsecond * delay * 1000) // ~700 Hz
		}
	}()

	for running {
		if emu.DrawFlag {
			ui.Draw(emu.Screen)
			emu.DrawFlag = false
		}
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break
			case *sdl.KeyboardEvent:
				if t.Keysym.Sym == sdl.K_ESCAPE {
					running = false
				}

				keyEventType := event.GetType()
				k, ok := keyMap[int(t.Keysym.Sym)]
				if !ok {
					continue
				}
				if keyEventType == sdl.KEYDOWN {
					emu.KeyDown(k)
				} else if keyEventType == sdl.KEYUP {
					emu.KeyUp(k)
				}
			}
		}
		time.Sleep(time.Microsecond * 16700)
	}
}

func getKeyMap() map[int]uint8 {
	keyMap = make(map[int]uint8)
	keyMap[sdl.K_1] = 0x1
	keyMap[sdl.K_2] = 0x2
	keyMap[sdl.K_3] = 0x3
	keyMap[sdl.K_4] = 0xc

	keyMap[sdl.K_q] = 0x4
	keyMap[sdl.K_w] = 0x5
	keyMap[sdl.K_e] = 0x6
	keyMap[sdl.K_r] = 0xd

	keyMap[sdl.K_a] = 0x7
	keyMap[sdl.K_s] = 0x8
	keyMap[sdl.K_d] = 0x9
	keyMap[sdl.K_f] = 0xe

	keyMap[sdl.K_z] = 0xa
	keyMap[sdl.K_x] = 0x0
	keyMap[sdl.K_c] = 0xb
	keyMap[sdl.K_v] = 0xf
	return keyMap
}
