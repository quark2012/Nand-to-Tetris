// This file is part of www.nand2tetris.org
// and the book "The Elements of Computing Systems"
// by Nisan and Schocken, MIT Press.
// File name: projects/04/Fill.asm

// Runs an infinite loop that listens to the keyboard input.
// When a key is pressed (any key), the program blackens the screen,
// i.e. writes "black" in every pixel;
// the screen should remain fully black as long as the key is pressed. 
// When no key is pressed, the program clears the screen, i.e. writes
// "white" in every pixel;
// the screen should remain fully clear as long as no key is pressed.

// Put your code here.

(KEYBOARD)
@KBD
D=M
@current_key // current_key = RAM[KBD]
M=D
@KEY
D;JNE // if RAM[KBD] != 0 goto KEY

@draw // draw = 0 (white)
M=0
@DRAWING
0;JMP

(KEY)
@draw // draw = -1 (black)
M=-1
@DRAWING
0;JMP

(DRAWING)
@i // i = 0
M=0

@8192 // max = 8192
D=A
@max
M=D

@SCREEN // addr = SCREEN
D=A
@addr
M=D

(LOOP)
@i // if (i>max) goto DONE
D=M
@max
D=D-M
@DONE
D;JGE

@draw // RAM[addr] = draw
D=M
@addr
A=M
M=D

@addr // addr = addr + 1
M=M+1
@i // i = i + 1
M=M+1

@LOOP
0;JMP

(DONE)
@KBD
D=M
@current_key
D=D-M
@KEYBOARD
D;JNE // if RAM[KBD] != current_key, goto KEYBOARD
@DONE
0;JMP
