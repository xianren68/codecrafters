package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// DirectOpt 重定向配置
type DirectOpt struct {
	isAppend bool
	isStdErr bool
	path     string
}

var osSystem = os.Getenv("OS")

type Handler = func([]string) (string, error)

var handlerMap map[string]Handler

func main() {
	// 保存当前终端状态
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("Failed to make raw: %v\n", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	handlerMap = make(map[string]Handler, 10)
	handlerMap["exit"] = handleExit
	handlerMap["echo"] = handleEcho
	handlerMap["type"] = handleType
	handlerMap["cd"] = handleCd
	handlerMap["pwd"] = handlePwd
	var args []string
	var reList []*DirectOpt
	out := bufio.NewWriter(os.Stdout)
	for {
		_, _ = out.WriteString("$ ")
		out.Flush()
		args, reList, err = ParseArgs()
		if err != nil {
			if err == io.EOF {
				break
			}
			out.WriteString(err.Error() + "\r\n")
			out.Flush()
			continue
		}
		if len(args) == 0 {
			outPut("", "", out, reList)
			continue
		}
		command := args[0]
		var str string
		if handler, ok := handlerMap[command]; ok {
			str, err = handler(args[1:])
			if err == nil {
				err = errors.New("")
			}
			outPut(str, err.Error(), out, reList)
		} else if flag, fullPath := isExecutable(command); flag {
			str, err = execution(fullPath, args[1:])
			if err == nil {
				err = errors.New("")
			}
			outPut(str, err.Error(), out, reList)
		} else {
			str = fmt.Sprintf("%s: command not found\r\n", command)
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
		return "", nil
	}
	return strings.Join(args, " ") + "\r\n", nil
}

func handleType(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	res := strings.Builder{}
	for _, val := range args {
		if _, ok := handlerMap[val]; ok {
			res.WriteString(fmt.Sprintf("%s is a shell builtin\r\n", val))
		} else if flag, cp := isExecutable(val); flag {
			res.WriteString(fmt.Sprintf("%s is %s\r\n", val, cp))
		} else {
			res.WriteString(fmt.Sprintf("%s: not found\r\n", val))
		}
	}
	return res.String(), nil
}

func handlePwd(args []string) (string, error) {
	_ = args
	s, err := os.Getwd()
	if err != nil {
		err = errors.New(err.Error() + "\r\n")
	}
	return s + "\r\n", err
}

func handleCd(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	if len(args) > 1 {
		return "", errors.New("cd: too many arguments")
	}
	homeVar := "HOME"
	if osSystem == "Windows_NT" {
		homeVar = "USERPROFILE"
	}
	if args[0] == "~" {
		args[0] = os.Getenv(homeVar)
	}
	err := os.Chdir(args[0])
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("cd: " + args[0] + ": No such file or directory")
	}
	return "", err
}
func autoComplete(args string) string {
	if len(args) < 3 {
		return ""
	}
	for key := range handlerMap {
		if strings.HasPrefix(key, args) {
			return key
		}
	}
	return ""
}

// ParseArgs 解析参数
func ParseArgs() ([]string, []*DirectOpt, error) {
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
	var reader = os.Stdin
	var out = bufio.NewWriter(os.Stdout)
	// 解析结果
	res := make([]string, 0, 20)
	buf := make([]byte, 1)
	for {
		_, err := reader.Read(buf)
		if err != nil {
			return res, optList, err
		}
		val := buf[0]
		if val == '\r' {
			fmt.Print("\n")
			break
		}
		if val == '\n' {
			fmt.Print("\r\n")
			break
		}
		if val == '\t' {
			val := string(item)
			str := autoComplete(val)
			fmt.Print(str[len(val):] + " ") // 输出补全部分
			res = append(res, str)
			item = make([]byte, 0, 1024)
			continue
		}
		out.WriteByte(val) // 输出到标准输出
		out.Flush()
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
			if val == '\'' {
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
				inDoubleQuote = true
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
	if len(item) == 0 && Redirect != nil {
		return nil, nil, errors.New("syntax error near unexpected token `newline'")
	}
	if len(item) != 0 {
		// if osSystem == "Windows_NT" {
		// 	item = item[:len(item)-1]
		// }
		if Redirect != nil {
			optList = append(optList, Redirect)
			Redirect.path = string(item)
		} else {
			res = append(res, string(item))
		}
	}
	return res, optList, nil
}

// 重定向
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

// 输出
func outPut(val string, errVal string, out *bufio.Writer, reList []*DirectOpt) {
	if errVal != "" {
		errVal = strings.ReplaceAll(errVal, "\n", "\r\n")
		if errVal[len(errVal)-1] != '\n' {
			errVal += "\r\n"
		}
	}
	if val != "" {
		val = strings.ReplaceAll(val, "\n", "\r\n")
		if val[len(val)-1] != '\n' {
			val += "\r\n"
		}
	}
	stdOutFlag := false
	stdErrFlag := false
	var err error
	for _, opt := range reList {
		if opt.isStdErr {
			stdErrFlag = true
			err = reToFile(errVal, opt.path, opt.isAppend)
		} else {
			stdOutFlag = true
			err = reToFile(val, opt.path, opt.isAppend)
		}
		if err != nil {
			out.WriteString(err.Error() + "\r\n")
			out.Flush()
		}
	}
	if !stdOutFlag {
		out.WriteString(val)
	}
	if !stdErrFlag {
		out.WriteString(errVal)
	}
	out.Flush()
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

func isExecutable(command string) (bool, string) {
	// 获取环境变量
	pathEnv := os.Getenv("PATH")
	split := ":"
	suffix := ""
	if osSystem == "Windows_NT" {
		split = ";"
		suffix = ".exe"
	}
	paths := strings.Split(pathEnv, split)
	for _, path := range paths {
		completePath := path + "/" + command + suffix
		info, err := os.Stat(completePath)
		if err != nil || info.IsDir() {
			continue
		}
		notExec := false
		if osSystem != "Windows_NT" {
			notExec = info.Mode()&0o111 == 0
		}
		if notExec {
			continue
		}
		return true, completePath
	}
	return false, ""
}

// 执行可执行文件
func execution(command string, args []string) (string, error) {
	commands := strings.Split(command, "/")
	if len(commands) > 1 {
		command = commands[len(commands)-1]
	}
	cmd := exec.Command(command, args...)
	var stdStr, errStr bytes.Buffer
	cmd.Stdout = &stdStr
	cmd.Stderr = &errStr
	err := cmd.Run()
	if err != nil {
		return stdStr.String(), errors.New(errStr.String())
	}
	return stdStr.String(), nil
}
