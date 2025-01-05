use std::collections::HashMap;
use std::process;
use std::sync::OnceLock;
use::std::fmt::Write as FmtWrite;
use std::io::{self, BufReader, BufWriter, BufRead, Write};
type Handler = fn(&[String]) -> Result<String, String>;
static GLOBAL_MAP: OnceLock<HashMap<String, Handler>> = OnceLock::new();
fn init_map() -> &'static HashMap<String, Handler> {
    GLOBAL_MAP.get_or_init(|| {
        let mut map:HashMap<String, Handler> = HashMap::new();
        map.insert("exit".to_string(), handle_exit);
        map.insert("echo".to_string(), handle_echo);
        map.insert("type".to_string(), handle_type);
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
        match parse_args(line) {
            Ok(args) => {
                if args.is_empty() {
                    continue;
                }
                if map.contains_key(&args[0]) {
                    let handler = map.get(&args[0]).unwrap();
                    let val = handler(&args[1..]).unwrap();
                    writer.write_fmt(format_args!("{}",val)).unwrap();
                }else {
                    writer.write_fmt(format_args!("{}: command not found\n", args[0])).unwrap();
                    writer.flush().unwrap();
                }
            },
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
    let res = args.join(" ");
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
            res.write_str(&*(arg.to_owned() + " is shell builtin\n")).unwrap();
        } else {
            res.write_str(&*(arg.to_owned() + ": not found\n")).unwrap();
        }
    }
    Ok(res)
}

fn parse_args(args: String) -> Result<Vec<String>, String> {
    let mut res = Vec::<String>::new();
    let mut in_single_quote = false;
    let mut in_double_quote = false;
    let mut escape_next = false;
    let mut item = String::new();
    for char in args.chars() {
        if in_single_quote {
            if char == '\'' {
                in_single_quote = false;
            }else {
                item.push(char);
            }
        }else if in_double_quote {
            if escape_next {
                escape_next = false;
                match char {
                    '$'|'"'|'\\' => item.push(char),
                    _ => {
                        item.push('\\');
                        item.push(char);
                    }
                }
            }else if char == '"' {
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
                '"'=> in_double_quote = true,
                '\''=> in_single_quote = true,
                '\\' => escape_next = true,
                ' ' => {
                    if item.len() != 0 {
                        res.push(item.clone());
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
        res.push(item.clone());
    }
    Ok(res)
}

