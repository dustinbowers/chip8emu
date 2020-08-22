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

/*
Memory Map:
+---------------+= 0xFFF (4095) End of Chip-8 RAM
|               |
|               |
|               |
|               |
|               |
| 0x200 to 0xFFF|
|     Chip-8    |
| Program / Data|
|     Space     |
|               |
|               |
|               |
+- - - - - - - -+= 0x600 (1536) Start of ETI 660 Chip-8 programs
|               |
|               |
|               |
+---------------+= 0x200 (512) Start of most Chip-8 programs
| 0x000 to 0x1FF|
| Reserved for  |
|  interpreter  |
+---------------+= 0x000 (0) Start of Chip-8 RAM

*/

type Chip8 struct {
	Screen   [64][32]uint8 // flags for pixel on or off
	Memory   [4096]byte    // Program entry point is typically 0x200
	V        [16]byte      // 16 8-bit registers (note VF is a carry-flag register)
	PC       uint16        // Program/Instruction counter
	I        uint16        // Index register
	SP       uint16        // Stack pointer
	Stack    [16]uint16
	DT       uint8 // Delay timer
	ST       uint8 // Sound timer
	DrawFlag bool  // Causes a redraw when set

	/*
		Input: 16 keys, 0 to F (8, 4, 6, 2 are used for direction input)
		1	2	3	C
		4	5	6	D
		7	8	9	E
		A	0	B	F
	*/
	keyboard [16]bool // Keys range from 0-F in a 4x4 grid

	// internals for easier opcode processing
	lastKey     *uint8 // Used for interrupting an input block (see Fx0A - LD Vx, K below)
	opcode      uint16 // Stores the current opcode. All opcodes are 2 bytes
	x, y, n, kk uint8  // various parts of the current opcode, used for easier processing
	nnn         uint16 // Stores addresses from opcodes
}

func (ch *Chip8) Initialize() {
	// Load fontset into memory (16 8x5 sprites)
	for i, b := range fontSet {
		ch.Memory[i+0x050] = b
	}

	// Set Entrypoint
	ch.PC = 0x200

	// Start subroutine for Delay timer and Sound Timer
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

func (ch *Chip8) EmulateCycle() (bool, error) {
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
	ch.opcode = (uint16(pcByte) << 8) | uint16(pc1Byte)

	// These internal values are always calculated, but not always used
	// 0000 0000 0000 0000
	//      x--- y--- n---
	//      nnn-----------
	//           kk-------
	ch.n = pc1Byte & 0x0F        // lower 4 bits of low byte
	ch.x = pcByte & 0x0F         // lower 4 bits of high byte
	ch.y = (pc1Byte >> 4) & 0x0F // upper 4 bits of low byte
	ch.kk = pc1Byte
	ch.nnn = ch.opcode & 0x0FFF

	ch.PC += 2 // Advance the program counter after we have the internals set for processing
}

func (ch *Chip8) executeOpcode() error {

	// Opcode table reference: https://en.wikipedia.org/wiki/CHIP-8#Opcode_table

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
		ch.V[0xF] = 0 // reset carry flag
		for byteInd := 0; byteInd < int(ch.n); byteInd++ {
			spriteByte := ch.Memory[int(ch.I)+byteInd]
			for bitInd := 0; bitInd < 8; bitInd++ {
				bit := (spriteByte >> bitInd) & 0x1

				screenX := (col + byte(7-bitInd)) % 64
				screenY := (row + byte(byteInd)) % 32

				currVal := ch.Screen[screenX][screenY]
				if bit == 1 && currVal == 1 {
					ch.V[0xF] = 1 // set carry flag if a collision occurs
				}

				ch.Screen[screenX][screenY] ^= bit // toggle pixels
			}
		}
		ch.DrawFlag = true // need a redraw

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
			fmt.Print("Waiting for keypress ")
			for {
				if ch.lastKey == nil {
					time.Sleep(time.Microsecond * 1600) // ~700 Hz
					continue
				}
				ch.V[ch.x] = *ch.lastKey
				fmt.Println("Got a keypress", ch.V[ch.x])
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
			ch.I = uint16(ch.V[ch.x])*5 + 0x050
		case 0x33: // Fx33 - LD B, Vx
			ch.Memory[ch.I] = uint8((uint16(ch.V[ch.x]) % 1000) / 100) // Hundreds place
			ch.Memory[ch.I+1] = (ch.V[ch.x] % 100) / 10                // Tens place
			ch.Memory[ch.I+2] = ch.V[ch.x] % 10                        // Ones place
		case 0x55: // Fx55 - LD [I], Vx
			for a := 0; a <= int(ch.x); a++ {
				ch.Memory[ch.I+uint16(a)] = ch.V[a]
			}
			ch.I += uint16(ch.x) + 1
		case 0x65: // Fx65 - LD Vx, [I]
			for a := 0; a <= int(ch.x); a++ {
				ch.V[a] = ch.Memory[ch.I+uint16(a)]
			}
			ch.I += uint16(ch.x) + 1
		default:
			return fmt.Errorf("unknown opcode: %x", ch.opcode)
		}
	}
	return nil
}

func (ch *Chip8) KeyDown(key uint8) {
	ch.lastKey = &key
	ch.keyboard[key] = true
}

func (ch *Chip8) KeyUp(key uint8) {
	ch.keyboard[key] = false
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
