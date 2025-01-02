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
	isStdErr bool
	path     string
}

type Handler = func([]string) (string, error)

var handlerMap map[string]Handler

func main() {
	handlerMap = make(map[string]Handler, 10)
	handlerMap["exit"] = handleExit
	handlerMap["echo"] = handleEcho
	handlerMap["type"] = handleType
	scanner := bufio.NewScanner(os.Stdin)
	var err error
	var args []string
	var reList []*DirectOpt
	out := bufio.NewWriter(os.Stdout)
	for {
		_, _ = out.WriteString("$ ")
		out.Flush()
		if !scanner.Scan() {
			return
		}
		line := scanner.Text()
		args, reList, err = ParseArgs(line)
		if err != nil {
			out.WriteString(err.Error() + "\n")
			out.Flush()
			continue
		}
		if len(args) == 0 {
			outPut("", nil, out, reList)
			continue
		}
		command := args[0]
		var str string
		if handler, ok := handlerMap[command]; ok {
			str, err = handler(args[1:])
			outPut(str, err, out, reList)
		} else {
			str = fmt.Sprintf("%s: command not found\n", command)
			out.WriteString(str)
			out.Flush()
		}
	}
}

func handleExit(args []string) (string, error) {
	if len(args) == 0 {
		os.Exit(0)
	}
	status := args[0]
	code, err := strconv.Atoi(status)
	if err != nil {
		code = 2
	}
	os.Exit(code)
	return "", nil
}

func handleEcho(args []string) (string, error) {
	if len(args) == 0 {
		return "\n", nil
	}
	return strings.Join(args, " ") + "\n", nil
}

func handleType(args []string) (string, error) {
	if len(args) == 0 {
		return "\n", nil
	}
	var err error
	res := strings.Builder{}
	for _, val := range args {
		if _, ok := handlerMap[val]; ok {
			res.WriteString(fmt.Sprintf("%s is a shell builtin\n", val))
		} else {
			err = fmt.Errorf("%s: not found\n", val)
		}
	}
	return res.String(), err
}

// ParseArgs 解析参数
func ParseArgs(args string) ([]string, []*DirectOpt, error) {
	// 是否单引号内
	inSingalQuote := false
	// 是否双引号内
	inDoubleQuote := false
	// 是否为转义字符后
	escapeNext := false
	// 解析出的每一项
	item := make([]byte, 0, 1024)
	// 是否重定向
	var Redirect *DirectOpt = nil
	var optList []*DirectOpt = nil
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
					if Redirect != nil {
						optList = append(optList, Redirect)
						Redirect.path = string(item)
						Redirect = nil
						item = make([]byte, 0, 1024)
						continue
					}
					Redirect = isRedirect(string(item))
					if Redirect == nil {
						res = append(res, string(item))
					}
					item = make([]byte, 0, 1024)
				}
			default:
				item = append(item, val)
			}
		}
	}
	if inSingalQuote || inDoubleQuote {
		return nil, nil, errors.New("unclosed quote")
	}
	if len(item) != 0 {
		if Redirect != nil {
			optList = append(optList, Redirect)
			Redirect.path = string(item)
		} else {
			res = append(res, string(item))
		}
	}
	return res, optList, nil
}

func isRedirect(val string) *DirectOpt {
	switch val {
	case ">", "1>":
		return &DirectOpt{}
	case ">>", "1>>":
		return &DirectOpt{isAppend: true}
	case "2>":
		return &DirectOpt{isStdErr: true}
	case "2>>":
		return &DirectOpt{isAppend: true, isStdErr: true}
	default:
		return nil
	}
}

func outPut(val string, errVal error, out *bufio.Writer, reList []*DirectOpt) {
	if len(reList) == 0 {
		if errVal != nil {
			out.WriteString(errVal.Error())
		} else {
			out.WriteString(val)
		}
		out.Flush()
		return
	}
	if errVal == nil {
		errVal = errors.New("")
	}
	var err error
	for _, opt := range reList {
		if opt.isStdErr {
			err = reToFile(errVal.Error(), opt.path, opt.isAppend)
		} else {
			err = reToFile(val, opt.path, opt.isAppend)
		}
		if err != nil {
			out.WriteString(err.Error())
			out.WriteByte('\n')
			out.Flush()
		}
	}
}

// 重定向到文件
func reToFile(val, path string, isAppend bool) error {
	if path == "" {
		return errors.New("syntax error near unexpected token `newline'")
	}
	var file *os.File
	var err error
	flag := os.O_RDWR | os.O_CREATE
	if isAppend {
		flag |= os.O_APPEND
	}
	file, err = os.OpenFile(path, flag, 0644)
	if err != nil {
		return err
	}
	file.WriteString(val)
	return file.Close()
}
