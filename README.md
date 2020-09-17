# Chip8 Emulator

A super basic emulation of [CHIP8](https://en.wikipedia.org/wiki/CHIP-8). Nothin' fancy here

<img src="https://github.com/dustinbowers/chip8emu/blob/master/screens/space-invaders.png" width="50%">

`roms/` may or may not be available here https://github.com/jamesmcm/chip8go/tree/master/roms

## Usage

Install some stuff if you don't have SDL packages yet 
```
brew install sdl2{,_image,_mixer,_ttf,_gfx} pkg-config
```

- Build: `make`
    - `./build/chip8-darwin [rom path]`
- Run: `make run`

<sub>(Or live dangerously and run the pre-compiled darwin binary in `build/`)</sub>

## Input

| Key | Description                             |
|-----|-----------------------------------------|
|  p  | Pause emulator processing               |
|  o  | Resume emulator processing              |
|  i  | Inspect state of emulator (see console) |

**Gamepad input:** 16 keys, 0 to F (8, 4, 6, 2 are sometimes used for direction input)

###### Original gamepad
```
        1    2    3    C
        4    5    6    D
        7    8    9    E
        A    0    B    F
```

###### Keyboard mapping to the above ^
```
        1    2    3    4
        Q    W    E    R
        A    S    D    F
        Z    X    C    V
```

## Architecture basics

### Registers
- `V0`-`VF`: 16 * 1 byte

**Note:** `VF` is a "carry flag" register, used for carry / non-borrow / collision detection


### Opcode table

There are 35 opcodes, each 2 bytes wide.

See: https://en.wikipedia.org/wiki/CHIP-8#Opcode_table

### Memory map

```
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
```

### Stack
Typically the stack is 48 bytes (12 levels * 4 address bits each). This implementation supports 16 levels

### Timers (60hz)

- Delay timer (DT)
- Sound timer (ST) 
    
## TODO

_Note: I'm no longer actively working on this project_

- [x] File-path argument
- [x] Audio beep
- [x] Debug outputs
- [ ] RESET key
- [ ] Tests
- [ ] (maybe?) Double buffering to prevent flickering
- [ ] SUPER CHIP8 opcodes
- [ ] Hires mode

## References

- https://en.wikipedia.org/wiki/CHIP-8
- http://devernay.free.fr/hacks/chip8/C8TECH10.HTM
- [Chip8.pdf](docs/Chip8.pdf)
- Special thanks to https://github.com/jamesmcm/chip8go/ for `roms/` collection

## Other screens

<img src="https://github.com/dustinbowers/chip8emu/blob/master/screens/pong.png" width="50%">

<img src="https://github.com/dustinbowers/chip8emu/blob/master/screens/ibm-logo.png" width="50%">
