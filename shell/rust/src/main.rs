use ::std::fmt::Write as FmtWrite;
use std::collections::HashMap;
use std::io::{self, BufRead, BufReader, BufWriter, Write};
use std::path::Path;
use std::sync::OnceLock;
use std::{env, fs, process};

#[cfg(target_os = "linux")]
use std::os::unix::fs::PermissionsExt;
#[cfg(target_os = "windows")]
mod system {
    pub const SPLIT_SYMBOL: &'static str = ";";
    pub const SUFFIX: &'static str = ".exe";
}
#[cfg(target_os = "linux")]
mod system {
    pub const SPLIT_SYMBOL: &'static str = ":";
    pub const SUFFIX: &'static str = "";
}

type Handler = fn(&[String]) -> Result<String, String>;
static GLOBAL_MAP: OnceLock<HashMap<String, Handler>> = OnceLock::new();
struct RedirectOpt {
    is_stderr: bool,
    is_append: bool,
    path: String,
}
fn init_map() -> &'static HashMap<String, Handler> {
    GLOBAL_MAP.get_or_init(|| {
        let mut map: HashMap<String, Handler> = HashMap::new();
        map.insert("exit".to_string(), handle_exit);
        map.insert("echo".to_string(), handle_echo);
        map.insert("type".to_string(), handle_type);
        map.insert("cd".to_string(), handle_cd);
        map.insert("pwd".to_string(), handle_pwd);
        map
    })
}
fn main() {
    let map = init_map();
    let mut reader = BufReader::new(io::stdin());
    let mut writer = BufWriter::new(io::stdout());
    loop {
        writer.write_fmt(format_args!("$ ")).unwrap();
        writer.flush().unwrap();
        let mut line = String::new();
        reader.read_line(&mut line).unwrap();
        line.pop().unwrap();
        if cfg!(target_os = "windows") {
            line.pop().unwrap();
        }
        match parse_args(line) {
            Ok(res) => {
                let args = res.0;
                let _ = res.1;
                if args.is_empty() {
                    continue;
                }
                if map.contains_key(&args[0]) {
                    let handler = map.get(&args[0]).unwrap();
                    let val = handler(&args[1..]).unwrap();
                    writer.write_fmt(format_args!("{}", val)).unwrap();
                } else {
                    writer
                        .write_fmt(format_args!("{}: command not found\n", args[0]))
                        .unwrap();
                    writer.flush().unwrap();
                }
            }
            Err(err) => {
                writer.write_fmt(format_args!("{}\n", err)).unwrap();
            }
        }
    }
}

fn handle_exit(args: &[String]) -> Result<String, String> {
    if args.len() == 0 {
        process::exit(0);
        return Ok("".to_string());
    }
    let status = args[0].to_string().parse::<i32>().unwrap_or(0);
    process::exit(status);
    Ok("".to_string())
}

fn handle_echo(args: &[String]) -> Result<String, String> {
    if args.len() == 0 {
        return Ok("".to_string());
    }
    let res = args.join(" ") + "\n";
    Ok(res)
}

fn handle_type(args: &[String]) -> Result<String, String> {
    if args.len() == 0 {
        return Ok("".to_string());
    }
    let map = init_map();
    let mut res = String::new();
    for arg in args {
        if map.contains_key(arg) {
            res.write_str(&*(arg.to_owned() + " is shell builtin\n"))
                .unwrap();
        } else {
            let (flag, complete_path) = is_executable(arg);
            if flag {
                res.write_str(&*(arg.to_owned() + " is " + &*complete_path + "\n")).unwrap();
                continue;
            }
            res.write_str(&*(arg.to_owned() + ": not found\n")).unwrap();
        }
    }
    Ok(res)
}

fn handle_pwd(args: &[String]) -> Result<String, String> {
    match env::current_dir() {
        Ok(dir) => Ok(dir.into_os_string().into_string().unwrap() + "\n"),
        Err(err) => Err(err.to_string() + "\n"),
    }
}

fn handle_cd(args: &[String]) -> Result<String, String> {
    if args.len() == 0 {
        return Ok("".to_string());
    }
    if args.len() > 1 {
        return Err("cd: too many arguments".to_string());
    }
    if args[0] == "~" {
        let sys = if cfg!(target_os = "windows") {
            "USERPROFILE"
        } else {
            "HOME"
        };
        let path_str = env::var(sys).unwrap();
        let path = Path::new(&path_str);
        return match env::set_current_dir(path) {
            Ok(_) => Ok("".to_string()),
            Err(err) => Err(err.to_string()),
        };
    }
    let path = Path::new(args[0].as_str());
    match env::set_current_dir(path) {
        Ok(_) => Ok("".to_string()),
        Err(err) => Err(err.to_string()),
    }
}

fn parse_args(args: String) -> Result<(Vec<String>, Vec<RedirectOpt>), String> {
    let mut res = Vec::<String>::new();
    let mut opts = Vec::<RedirectOpt>::new();
    let mut opt: Option<RedirectOpt> = None;
    let mut in_single_quote = false;
    let mut in_double_quote = false;
    let mut escape_next = false;
    let mut item = String::new();
    for char in args.chars() {
        if in_single_quote {
            if char == '\'' {
                in_single_quote = false;
            } else {
                item.push(char);
            }
        } else if in_double_quote {
            if escape_next {
                escape_next = false;
                match char {
                    '$' | '"' | '\\' => item.push(char),
                    _ => {
                        item.push('\\');
                        item.push(char);
                    }
                }
            } else if char == '"' {
                in_single_quote = true;
            } else if char == '\\' {
                escape_next = true;
            } else {
                item.push(char);
            }
        } else if escape_next {
            escape_next = false;
            item.push(char);
        } else {
            match char {
                '"' => in_double_quote = true,
                '\'' => in_single_quote = true,
                '\\' => escape_next = true,
                ' ' => {
                    if item.len() != 0 {
                        if let Some(mut reOpt) = opt {
                            reOpt.path = item.clone();
                            item.clear();
                            opt = None;
                            opts.push(reOpt);
                            continue;
                        }
                        opt = is_redirect(item.as_str());
                        if opt.is_none() {
                            res.push(item.clone());
                        }
                        item.clear();
                    }
                }
                _ => {
                    item.push(char);
                }
            }
        }
    }
    if in_single_quote || in_double_quote {
        return Err("unclosed quotes".to_string());
    }
    if item.len() != 0 {
        if let Some(mut opt) = opt {
            opt.path = item.clone();
            opts.push(opt);
        } else {
            res.push(item.clone());
        }
    }
    Ok((res, opts))
}

fn is_redirect(flag: &str) -> Option<RedirectOpt> {
    match flag {
        ">" | "1>" => Some(RedirectOpt {
            is_stderr: false,
            is_append: false,
            path: "".to_string(),
        }),
        ">>" | "1>>" => Some(RedirectOpt {
            is_stderr: false,
            is_append: true,
            path: "".to_string(),
        }),
        "2>" => Some(RedirectOpt {
            is_stderr: true,
            is_append: false,
            path: "".to_string(),
        }),
        "2>>" => Some(RedirectOpt {
            is_stderr: true,
            is_append: true,
            path: "".to_string(),
        }),
        _ => None,
    }
}

fn is_executable(flag: &str) -> (bool, String) {
    // 获取环境变量
    let paths = env::var("PATH").unwrap();
    let path_splits = paths.split(system::SPLIT_SYMBOL).collect::<Vec<&str>>();
    for path in path_splits {
        let complete_path = format!("{}/{}{}", path, &flag, system::SUFFIX);
        let file_opt = fs::metadata(&complete_path);
        if let Ok(file_info) = file_opt {
            if file_info.is_dir() {
                continue;
            }
            #[cfg(target_os = "windows")]
            return (true, complete_path.to_string());
            #[cfg(not(target_os = "windows"))]
            {
                let permissions = file_info.permissions();
                if permissions.mode() & 0o111 == 0 {
                    continue;
                }
                return (true, complete_path.to_string());
            }
        } else {
            continue;
        }
    }
    (false, "".to_string())
}
