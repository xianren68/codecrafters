package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// 重定向配置
type DirectOpt struct {
	isAppend bool
	isStderr bool
	path     string
}

type Handler = func([]string) string

var handlerMap map[string]Handler

func main() {
	handlerMap = make(map[string]Handler, 10)
	handlerMap["exit"] = handleExit
	handlerMap["echo"] = handleEcho
	handlerMap["type"] = handleType
	scanner := bufio.NewScanner(os.Stdin)
	var err error
	var args []string
	out := bufio.NewWriter(os.Stdout)
	for {
		_, _ = out.WriteString("$ ")
		out.Flush()
		if !scanner.Scan() {
			return
		}
		line := scanner.Text()
		args, err = ParseArgs(line)
		if err != nil {
			out.WriteString(err.Error() + "\n")
			out.Flush()
			continue
		}
		if len(args) == 0 {
			out.WriteString("\n")
			out.Flush()
			continue
		}
		command := args[0]
		var str string
		if handler, ok := handlerMap[command]; ok {
			str = handler(args[1:])
		} else {
			str = fmt.Sprintf("%s: command not found\n", command)
		}
		out.WriteString(str)
		out.Flush()
	}
}

func handleExit(args []string) string {
	if len(args) == 0 {
		os.Exit(0)
	}
	status := args[0]
	code, err := strconv.Atoi(status)
	if err != nil {
		code = 2
	}
	os.Exit(code)
	return ""
}

func handleEcho(args []string) string {
	if len(args) == 0 {
		return "\n"
	}
	return strings.Join(args, " ") + "\n"
}

func handleType(args []string) string {
	if len(args) == 0 {
		return "\n"
	}
	res := strings.Builder{}
	for _, val := range args {
		if _, ok := handlerMap[val]; ok {
			res.WriteString(fmt.Sprintf("%s is a shell builtin\n", val))
		} else {
			res.WriteString(fmt.Sprintf("%s: not found\n", val))
		}
	}
	return res.String()
}

func ParseArgs(args string) ([]string, error) {
	// 是否单引号内
	inSingalQuote := false
	// 是否双引号内
	inDoubleQuote := false
	// 是否为转义字符后
	escapeNext := false
	// 解析出的每一项
	item := make([]byte, 0, 1024)
	// 解析结果
	res := make([]string, 0, 20)
	for i := 0; i < len(args); i++ {
		val := args[i]
		if inDoubleQuote {
			if escapeNext {
				escapeNext = false
				switch val {
				case '"', '$', '\\':
					// 保留原值
					item = append(item, val)
				default:
					// 保留转义符
					item = append(item, '\\', val)
				}
			} else if val == '"' {
				inDoubleQuote = false
			} else if val == '\\' {
				escapeNext = true
			} else {
				item = append(item, val)
			}
		} else if inSingalQuote {
			if val == '\\' {
				escapeNext = true
			} else if val == '\'' {
				inSingalQuote = false
			} else {
				item = append(item, val)
			}
		} else if escapeNext {
			escapeNext = false
			item = append(item, val)
		} else {
			switch val {
			case '\\':
				escapeNext = true
			case '\'':
				inSingalQuote = true
			case '"':
				escapeNext = true
			case ' ':
				if len(item) > 0 {
					res = append(res, string(item))
					item = make([]byte, 0, 1024)
				}
			default:
				item = append(item, val)
			}
		}
	}
	if inSingalQuote || inDoubleQuote {
		return nil, errors.New("unclosed quote")
	}
	if len(item) != 0 {
		res = append(res, string(item))
	}
	return res, nil
}
