package chip8

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"
)

var fontSet = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

type Chip8 struct {
	Screen [64][32]uint8
	Memory [4096]byte
	V      [16]byte
	PC     uint16
	I      uint16
	SP     uint16
	Stack  [16]uint16
	DT     uint8
	ST     uint8
	DrawFlag bool

	// internals
	lastKey     *uint8
	keyboard    [16]bool
	opcode      uint16
	x, y, n, kk uint8
	nnn         uint16
}

func (ch *Chip8) Initialize() {
	// Load fontset into memory
	for i, b := range fontSet {
		ch.Memory[i+0x050] = b
	}

	// Entrypoint
	ch.PC = 0x200

	ch.startClock()

}

func (ch *Chip8) LoadRom(filepath string) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("loadRom: failed reading file: %v", err)
	}

	for i, b := range data {
		ch.Memory[i+0x200] = b
	}
	return nil
}

func (ch *Chip8) randScreen() { // for demo purposes
	for x, _ := range ch.Screen {
		for y, _ := range ch.Screen[x] {
			on := uint8(rand.Intn(2))
			ch.Screen[x][y] = on
		}
	}
}

func (ch *Chip8) EmulateCycle() (bool, error) {

	// Fetch opcode
	ch.fetchOpcode()

	// Decode & Execute opcode
	err := ch.executeOpcode()
	if err != nil {
		return false, err
	}

	return true, nil
}

func (ch *Chip8) fetchOpcode() {
	pcByte := ch.Memory[ch.PC]
	pc1Byte := ch.Memory[ch.PC+1]

	// Each opcode is 2 bytes
	// 0000 0000 0000 0000
	//      x--- y--- n---
	//      nnn-----------
	//           kk-------
	ch.opcode = (uint16(pcByte) << 8) | uint16(pc1Byte)
	ch.n = pc1Byte & 0x0F        // lower 4 bits of low byte
	ch.x = pcByte & 0x0F         // lower 4 bits of high byte
	ch.y = (pc1Byte >> 4) & 0x0F // upper 4 bits of low byte
	ch.kk = pc1Byte
	ch.nnn = ch.opcode & 0x0FFF

	ch.PC += 2
}

func (ch *Chip8) executeOpcode() error {

	switch ch.opcode & 0xF000 {
	case 0x0000:
		switch ch.kk {
		case 0x00E0: // 00E0 - CLS
			ch.Screen = [64][32]uint8{}
		case 0x00EE: // 00EE -  RET
			ch.PC = ch.Stack[ch.SP]
			ch.SP -= 1
		default:
			return fmt.Errorf("unknown opcode: %x", ch.opcode)
		}
	case 0x1000: // 1nnn - JP addr
		ch.PC = ch.nnn
	case 0x2000: // 2nnn - CALL addr
		ch.SP++
		ch.Stack[ch.SP] = ch.PC
		ch.PC = ch.nnn
	case 0x3000: // 3xkk - SE Vx, byte (skip if equal)
		if ch.V[ch.x] == ch.kk {
			ch.PC += 2
		}
	case 0x4000: // 4xkk - SNE Vx, byte (skip if not equal)
		if ch.V[ch.x] != ch.kk {
			ch.PC += 2
		}
	case 0x5000: // 5xy0 - SE Vx, Vy
		if ch.V[ch.x] == ch.V[ch.y] {
			ch.PC += 2
		}
	case 0x6000: // 6xkk - LD Vx, byte
		ch.V[ch.x] = ch.kk
	case 0x7000: // 7xkk - Add Vx, byte
		ch.V[ch.x] = ch.V[ch.x] + ch.kk
	case 0x8000: // Maths
		switch ch.n {
		case 0x0: // 8xy0 - LD Vx, Vy
			ch.V[ch.x] = ch.V[ch.y]
		case 0x1: // 8xy1 - OR Vx, Vy
			ch.V[ch.x] = ch.V[ch.x] | ch.V[ch.y]
		case 0x2: // 8xy2 - AND Vx, Vy
			ch.V[ch.x] = ch.V[ch.x] & ch.V[ch.y]
		case 0x3: // 8xy3 - XOR Vx, Vy
			ch.V[ch.x] = ch.V[ch.x] ^ ch.V[ch.y]
		case 0x4: // 8xy4 - ADD Vx, Vy
			if int16(ch.V[ch.x])+int16(ch.V[ch.y]) > 255 {
				ch.V[0xF] = 1
			} else {
				ch.V[0xF] = 0
			}
			ch.V[ch.x] = ch.V[ch.x] + ch.V[ch.y]
		case 0x5: // 8xy5 - SUB Vx, Vy
			if ch.V[ch.x] > ch.V[ch.y] {
				ch.V[0xF] = 1
			} else {
				ch.V[0xF] = 0
			}
			ch.V[ch.x] = ch.V[ch.x] - ch.V[ch.y]
		case 0x6: // 8xy6 - SHR Vx {, Vy}
			if ch.V[ch.x]&0x1 == 1 {
				ch.V[0xF] = 1
			} else {
				ch.V[0xF] = 0
			}
			ch.V[ch.x] = ch.V[ch.x] >> 1
		case 0x7: // 8xy7 - SUBN Vx, Vy
			if ch.V[ch.y] > ch.V[ch.x] {
				ch.V[0xF] = 1
			} else {
				ch.V[0xF] = 0
			}
			ch.V[ch.x] = ch.V[ch.y] - ch.V[ch.x]
		case 0xE: // 8xyE - SHL Vx {, Vy}
			if (ch.V[ch.x]>>7)&0x1 == 1 {
				ch.V[0xF] = 1
			} else {
				ch.V[0xF] = 0
			}
			ch.V[ch.x] = ch.V[ch.x] << 1
		default:
			return fmt.Errorf("unknown opcode: %x", ch.opcode)
		}
	case 0x9000: // 9xy0 - SNE Vx, Vy
		switch ch.n {
		case 0x0:
			if ch.V[ch.x] != ch.V[ch.y] {
				ch.PC += 2
			}
		default:
			return fmt.Errorf("unknown opcode: %x", ch.opcode)
		}
	case 0xA000: // Annn - LD I, addr
		ch.I = ch.nnn
	case 0xB000: // Bnnn - JP V0, addr
		ch.PC = uint16(ch.V[0x0]) + ch.nnn
	case 0xC000: // Cxkk - RND Vx, byte
		ch.V[ch.x] = uint8(rand.Intn(256)) & ch.kk
	case 0xD000: // Dxyn - DRW Vx, Vy, nibble
		col := ch.V[ch.x]
		row := ch.V[ch.y]
		ch.V[0xF] = 0
		for byteInd := 0; byteInd < int(ch.n); byteInd++ {
			spriteByte := ch.Memory[int(ch.I) + byteInd]
			for bitInd := 0; bitInd < 8; bitInd++ {
				bit := (spriteByte >> bitInd) & 0x1

				screenX := (col + byte(7 - bitInd)) % 64
				screenY := (row + byte(byteInd)) % 32

				currVal := ch.Screen[screenX][screenY]
				if bit == 1 && currVal == 1 {
					ch.V[0xF] = 1
				}

				ch.Screen[screenX][screenY] ^= bit // currVal != bool(bit)
			}
		}
		ch.DrawFlag = true

	case 0xE000: // User inputs
		switch ch.kk {
		case 0x9E: // Ex9E - SKP Vx
			if ch.keyboard[ch.V[ch.x]] {
				ch.PC += 2
			}
		case 0xA1: // ExA1 - SKNP Vx
			if ch.keyboard[ch.V[ch.x]] == false {
				ch.PC += 2
			}
		default:
			return fmt.Errorf("unknown opcode: %x", ch.opcode)
		}
	case 0xF000: // Misc stuffs
		switch ch.kk {
		case 0x07: // Fx07 - LD Vx, DT
			ch.V[ch.x] = ch.DT
		case 0x0A: // Fx0A - LD Vx, K
			fmt.Print("waiting for keypress ")
			for {
				if ch.lastKey == nil {
					//fmt.Print(".")
					time.Sleep(time.Microsecond * 1600) // ~700 Hz
					continue
				}
				ch.V[ch.x] = *ch.lastKey
				fmt.Println("GOT A KEYPRESS! ", ch.V[ch.x])
				ch.lastKey = nil
			}
		case 0x15: // Fx15 - LD DT, Vx
			ch.DT = ch.V[ch.x]
		case 0x18: // Fx18 - LD ST, Vx
			ch.ST = ch.V[ch.x]
		case 0x1E: // Fx1E - ADD I, Vx
			ch.I += uint16(ch.V[ch.x])

			// See: https://en.wikipedia.org/wiki/CHIP-8#cite_note-16
			//if ch.I > 0xFFF {
			//	ch.V[0xF] = 1
			//} else {
			//	ch.V[0xF] = 0
			//}
		case 0x29: // Fx29 - LD F, Vx
			ch.I = uint16(ch.V[ch.x]) * 5 + 0x050
		case 0x33: // Fx33 - LD B, Vx
			ch.Memory[ch.I] = uint8((uint16(ch.V[ch.x]) % 1000) / 100)
			ch.Memory[ch.I+1] = (ch.V[ch.x] % 100) / 10
			ch.Memory[ch.I+2] = ch.V[ch.x] % 10
		case 0x55: // Fx55 - LD [I], Vx
			for a := 0; a <= int(ch.x); a++ {
				ch.Memory[ch.I + uint16(a)] = ch.V[a]
			}
			ch.I += uint16(ch.x) + 1
		case 0x65: // Fx65 - LD Vx, [I]
			for a := 0; a <= int(ch.x); a++ {
				ch.V[a] = ch.Memory[ch.I + uint16(a)]
			}
			ch.I += uint16(ch.x) + 1
		default:
			return fmt.Errorf("unknown opcode: %x", ch.opcode)
		}
	}
	return nil
}

func (ch* Chip8) KeyDown(key uint8) {
	ch.lastKey = &key
	ch.keyboard[key] = true
	//fmt.Printf("\\/ Keyboard: %+v\n", ch.keyboard)
}

func (ch* Chip8) KeyUp(key uint8) {
	ch.keyboard[key] = false
	//fmt.Printf("/\\ Keyboard: %+v\n", ch.keyboard)
}

func (ch *Chip8) startClock() {
	go func() {
		for {
			ch.decrementTimers()
			time.Sleep(time.Microsecond * 16700) // Clock timers run at 60 Hz
		}
	}()
}

func (ch *Chip8) decrementTimers() {
	if ch.ST > 0 {
		ch.ST--
		if ch.ST == 0 {
			fmt.Println("=====BEEEEP=====")
		}
	}
	if ch.DT > 0 {
		ch.DT--
	}
}

func (ch *Chip8) unknownOpcode() {

}


