package main

import (
	"fmt"
	"os"
)

var tokenMap = map[byte]string{
	'(': "LEFT_PAREN",
	')': "RIGHT_PAREN",
	'{': "LEFT_BRACE",
	'}': "RIGHT_BRACE",
	'*': "STAR",
	'+': "PLUS",
	'.': "DOT",
	',': "COMMA",
	'-': "MINUS",
	'/': "SLASH",
	';': "SEMICOLON",
	'=': "EQUAL",
}

var errorMap = map[byte]struct{}{
	'$': {},
	'#': {},
	'@': {},
	'%': {},
}

var nextEqul = map[byte]string{
	'=': "EQUAL",
	'>': "GREATER",
	'<': "LESS",
	'!': "BANG",
}

var igonreMap = map[byte]struct{}{
	' ':  {},
	'\t': {},
	'\n': {},
}

func main() {
	args := os.Args
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "too few arguments")
		os.Exit(1)
	}
	command := args[1]
	if command != "tokenize" {
		fmt.Printf("unknow command %s\n", command)
		os.Exit(1)
	}
	fileName := args[2]
	file, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open file error %v", err)
		os.Exit(1)
	}
	if len(file) > 0 {
		handler(file)
	} else {
		fmt.Println("EOF null")
	}
}

func handler(content []byte) {
	token := make([]byte, 0, 20)
	line := 1
	expectFlag := false
	isComment := false
	for i := 0; i < len(content); i++ {
		val := content[i]
		if _, ok := igonreMap[val]; ok {
			if val == '\n' {
				line++
				isComment = false
			}
		} else if isComment {
			continue
		} else if val == '/' && i+1 < len(content) && content[i+1] == '/' {
			isComment = true
			i++
		} else if v, ok := tokenMap[val]; ok {
			str := "null"
			if len(token) > 0 {
				str = string(token)
			}
			if v1, ok := nextEqul[val]; ok && i+1 < len(content) && content[i+1] == '=' {
				fmt.Printf("%s_%s %c%c %s\n", v1, "EQUAL", val, '=', str)
				i++
			} else {
				fmt.Printf("%s %c %s\n", v, val, str)

			}
			token = token[:0]
		} else if _, ok := errorMap[val]; ok {
			fmt.Fprintf(os.Stderr, "[line %d] Error: Unexpected character: %c\n", line, val)
			expectFlag = true
			token = token[:0]
		} else {
			token = append(token, val)
		}
	}
	fmt.Println("EOF  null")
	if expectFlag {
		os.Exit(65)
	}
	os.Exit(0)
}
