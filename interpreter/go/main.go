package main

import (
	"fmt"
	"os"
)

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
	for _, val := range content {
		if val == '(' {
			str := "null"
			if len(token) > 0 {
				str = string(token)
			}
			fmt.Printf("LEFT_PAREN ( %s\n", str)
			token = make([]byte, 0, 20)
		} else if val == ')' {
			str := "null"
			if len(token) > 0 {
				str = string(token)
			}
			fmt.Printf("RIGHT_PAREN ) %s\n", str)
			token = make([]byte, 0, 20)
		} else {
			token = append(token, val)
		}
	}
	fmt.Println("EOF  null")
}
