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
	'>': "GREATER",
	'<': "LESS",
	'!': "BANG",
}

var errorMap = map[byte]struct{}{
	'$': {},
	'#': {},
	'@': {},
	'%': {},
}

var equlPre = map[byte]string{
	'=': "EQUAL",
	'>': "GREATER",
	'<': "LESS",
	'!': "BANG",
}

var ignoreMap = map[byte]struct{}{
	' ':  {},
	'\t': {},
}

func main() {
	if len(os.Args) < 3 {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: ./your_program.sh tokenize <filename>")
		os.Exit(1)
	}

	command := os.Args[1]

	if command != "tokenize" {
		_, _ = fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
	filename := os.Args[2]
	fileContents, err := os.ReadFile(filename)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	if len(fileContents) > 0 {
		handler(fileContents)
	} else {
		fmt.Println("EOF  null") // Placeholder, remove this line when implementing the scanner
	}
}
func handler(content []byte) {
	token := make([]byte, 0, 20)
	inIdentity := false
	inNumber := false
	inComment := false
	inSingleLine := false
	isDot := false
	expectFlag := false
	line := 1
	for i := 0; i < len(content); i++ {
		val := content[i]
		if inComment {
			if val == '\n' {
				inComment = false
				token = token[:0]
				line++
			}
		} else if inSingleLine {
			if val == '"' {
				inSingleLine = false
				fmt.Printf("STRING \"%s\" %s\n", string(token), string(token))
				token = token[:0]
			} else if val == '\n' {
				inSingleLine = false
				expectFlag = true
				_, _ = fmt.Fprintf(os.Stderr, "[line %d] Error: Unterminated string.\n", line)
				token = token[:0]
				line++
			} else {
				token = append(token, val)
			}
		} else if inNumber {
			if (val >= '0' && val <= '9') || (val == '.') {
				if val == '.' {
					isDot = true
				}
				token = append(token, val)
			} else {
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
				inNumber = false
				token = token[:0]
				i--
			}
		} else if inIdentity {
			if _, ok := ignoreMap[val]; ok || val == '\n' {
				inIdentity = false
				fmt.Printf("IDENTIFIER %s %s\n", string(token), "null")
				token = token[:0]
				if val == '\n' {
					line++
				}
			} else if _, ok := tokenMap[val]; ok {
				inIdentity = false
				fmt.Printf("IDENTIFIER %s %s\n", string(token), "null")
				token = token[:0]
				i--
			} else {
				token = append(token, val)
			}
		} else if _, ok := ignoreMap[val]; ok {
			continue
		} else if _, ok := errorMap[val]; ok {
			expectFlag = true
			_, _ = fmt.Fprintf(os.Stderr, "[line %d] Error: Unexpected character: %c\n", line, val)
		} else if v, ok := tokenMap[val]; ok {
			str := "null"
			if len(token) > 0 {
				str = string(token)
			}
			if v1, ok := equlPre[val]; ok && i+1 < len(content) && content[i+1] == '=' {
				fmt.Printf("%s_%s %c%c %s\n", v1, "EQUAL", val, '=', str)
				i++
			} else if val == '/' && i+1 < len(content) && content[i+1] == '/' {
				inComment = true
				i++
			} else {
				fmt.Printf("%s %c %s\n", v, val, str)
			}
		} else {
			if val == '"' {
				inSingleLine = true
			} else if val >= '0' && val <= '9' {
				inNumber = true
				token = append(token, val)
			} else if val >= 'a' && val <= 'z' || val >= 'A' && val <= 'Z' || val == '_' {
				inIdentity = true
				token = append(token, val)
			} else if val == '\n' {
				line++
				token = token[:0]
			} else {
				token = append(token, val)
			}
		}
	}
	if inSingleLine {
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
	if inIdentity {
		fmt.Printf("IDENTIFIER %s %s\n", string(token), "null")
	}
	fmt.Println("EOF  null")
	if expectFlag {
		os.Exit(65)
	}
	os.Exit(0)
}
