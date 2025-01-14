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
	for i := 0; i < len(content); i++ {
		val := content[i]
		if v, ok := tokenMap[val]; ok {
			str := "null"
			if len(token) > 0 {
				str = string(token)
			}
			if v == "EQUAL" && i+1 < len(content) && content[i+1] == '=' {
				v = "EQUAL_EQUAL"
				i++
				fmt.Printf("%s %c%c %s\n", v, val, val, str)
			} else {
				fmt.Printf("%s %c %s\n", v, val, str)
			}
			token = token[:0]
		} else if _, ok := errorMap[val]; ok {
			fmt.Fprintf(os.Stderr, "[line %d] Error: Unexpected character: %c\n", line, val)
			expectFlag = true
			token = token[:0]
		} else {
			if val == '\n' {
				line++
			}
			token = append(token, val)
		}
	}
	fmt.Println("EOF  null")
	if expectFlag {
		os.Exit(65)
	}
	os.Exit(0)
}
