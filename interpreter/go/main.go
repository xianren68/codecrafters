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
	inSignal := false
	inNumber := false
	isDot := false
	for i := 0; i < len(content); i++ {
		val := content[i]
		if val == '\n' {
			if inSignal {
				expectFlag = true
				fmt.Fprintf(os.Stderr, "[line %d] Error: Unterminated string.\n", line)
			}
			line++
			isComment = false
			token = token[:0]
		} else if inSignal {
			if val == '"' {
				inSignal = false
				fmt.Printf("STRING \"%s\" %s\n", string(token), string(token))
				token = token[:0]
			} else {
				token = append(token, val)
			}
		} else if inNumber {
			if (val >= '0' && val <= '9') || val == '.' {
				token = append(token, val)
				if val == '.' {
					isDot = true
				}
			} else {
				str := string(token)
				if !isDot {
					str += ".0"
				} else {
					j := len(str) - 1
					for ; str[j] != '.'; j-- {
						if str[j] == 0 {
							continue
						} else {
							break
						}
					}
					str = str[:j+1]
					if str[j] == '.' {
						str += "0"
					}
					isDot = false
				}
				fmt.Printf("NUMBER %s %s\n", string(token), str)
				inNumber = false
				token = token[:0]
				i--
			}
		} else if _, ok := igonreMap[val]; ok || isComment {
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
			if val == '"' {
				inSignal = true
				continue
			} else if val >= '0' && val <= '9' {
				inNumber = true
				token = token[:0]
				token = append(token, val)
				continue
			}
			token = append(token, val)
		}
	}
	if inSignal {
		expectFlag = true
		fmt.Fprintf(os.Stderr, "[line %d] Error: Unterminated string.\n", line)
	}
	if inNumber {
		str := string(token)
		if !isDot {
			str += ".0"
		} else {
			j := len(str) - 1
			for ; str[j] != '.'; j-- {
				if str[j] == '0' {
					continue
				} else {
					break
				}
			}
			str = str[:j+1]
			if str[j] == '.' {
				str += "0"
			}
			isDot = false
		}
		fmt.Printf("NUMBER %s %s\n", string(token), str)
	}
	fmt.Println("EOF  null")
	if expectFlag {
		os.Exit(65)
	}
	os.Exit(0)
}
