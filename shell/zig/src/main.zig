const std = @import("std");
const builtin = @import("builtin");

const Allocator = std.heap.page_allocator;
const handler = fn ([][]u8) anyerror![]u8;
const hashMap = std.StringHashMap(*const handler);
var handleMap: hashMap = undefined;
const redirectOpt = struct {
    isAppend: bool,
    isStderr: bool,
    path: []u8,
};
const SPLIT = if (builtin.os.tag == .linux) ":" else ";";
const SUFFIX = if (builtin.os.tag == .windows) ".exe" else "";
const HOME_ENV = if (builtin.os.tag == .windows) "USERPROFILE" else "HOME";
pub fn main() !void {
    handleMap = hashMap.init(Allocator);
    defer handleMap.deinit();
    try handleMap.put("exit", handle_exit);
    try handleMap.put("echo", handle_echo);
    try handleMap.put("type", handle_type);
    try handleMap.put("pwd", handle_pwd);
    try handleMap.put("cd", handle_cd);
    var writer = std.io.getStdOut().writer();
    var reader = std.io.getStdIn().reader();
    var buffer: [1024]u8 = undefined;
    while (true) {
        try writer.print("$ ", .{});
        const input = try reader.readUntilDelimiter(&buffer, '\n');
        if (input.len == 0) {
            continue;
        }
        if (parse_args(input[0 .. input.len - 1])) |resu| {
            var result = resu;
            const args = result.args;
            const re_opts = result.reList;
            if (args.len == 0) {
                continue;
            }
            const cmd = args[0];
            if (handleMap.get(cmd)) |handlerFn| {
                out_put(handlerFn(args[1..]), re_opts);
            } else {
                if (is_executable(args[0])) |com_path| {
                    out_put(execution(com_path, args[1..]), re_opts);
                } else {
                    try writer.print("{s} command not found\n", .{input});
                }
            }
            result.deinit();
        } else |err| {
            try writer.print("error: {any}\n", .{err});
        }
    }
}

fn handle_exit(args: [][]u8) ![]u8 {
    if (args.len == 0) {
        std.process.exit(0);
    }
    const status = std.fmt.parseInt(u8, args[0], 10) catch @as(u8, 0);
    std.process.exit(status);
}

fn handle_echo(args: [][]u8) ![]u8 {
    if (args.len == 0) {
        return "";
    }
    var result = try std.ArrayList(u8).initCapacity(Allocator, 200);
    defer result.deinit();
    for (args) |arg| {
        try result.appendSlice(arg);
        try result.append(' ');
    }
    _ = result.pop();
    try result.append('\n');
    return try result.toOwnedSlice();
}

fn handle_type(args: [][]u8) ![]u8 {
    if (args.len == 0) {
        return "";
    }
    var res = try std.ArrayList(u8).initCapacity(Allocator, 200);
    for (args) |arg| {
        if (handleMap.get(arg)) |_| {
            try res.appendSlice(arg);
            try res.appendSlice(" is a shell builtin\n");
        } else if (is_executable(arg)) |path| {
            try res.appendSlice(arg);
            try res.appendSlice(" is ");
            try res.appendSlice(path);
            try res.appendSlice("\n");
        } else {
            try res.appendSlice(arg);
            try res.appendSlice(" not found\n");
        }
    }
    return try res.toOwnedSlice();
}

fn handle_pwd(args: [][]u8) ![]u8 {
    _ = args;
    var cur_dir = std.fs.cwd();
    defer cur_dir.close();
    const path = try cur_dir.realpathAlloc(Allocator, ".");
    defer Allocator.free(path);
    var res = try Allocator.alloc(u8, path.len + 1);
    std.mem.copyForwards(u8, res, path);
    res[path.len] = '\n';
    return res;
}

fn handle_cd(args: [][]u8) ![]u8 {
    if (args.len == 0) {
        return "";
    }
    if (args.len > 1) {
        return error.TooManyArgs;
    }
    var path: []const u8 = args[0];
    if (std.mem.eql(u8, path, "~")) {
        const home_env = try std.process.getEnvMap(Allocator);
        path = home_env.get(HOME_ENV).?;
    }
    var dir = try std.fs.cwd().openDir(path, .{});
    defer dir.close();
    try dir.setAsCwd();
    return "";
}

const res_type = struct {
    args: [][]u8,
    reList: []redirectOpt,
    fn deinit(self: *res_type) void {
        for (self.reList) |re| {
            Allocator.free(re.path);
        }
        for (self.args) |arg| {
            Allocator.free(arg);
        }
        Allocator.free(self.args);
        Allocator.free(self.reList);
    }
};
fn parse_args(args: []u8) !res_type {
    var result = try std.ArrayList([]u8).initCapacity(Allocator, 200);
    defer result.deinit();
    var item = try std.ArrayList(u8).initCapacity(Allocator, 200);
    defer item.deinit();
    var in_single_quote = false;
    var in_double_quote = false;
    var escape_next = false;
    var reOpt: ?redirectOpt = null;
    var reList = try std.ArrayList(redirectOpt).initCapacity(Allocator, 20);
    defer reList.deinit();
    for (args) |c| {
        if (in_single_quote) {
            if (c == '\'') {
                in_single_quote = false;
            } else {
                try item.append(c);
            }
        } else if (in_double_quote) {
            if (escape_next) {
                escape_next = false;
                switch (c) {
                    '"', '$', '\\' => try item.append(c),
                    else => {
                        try item.append('\\');
                        try item.append(c);
                    },
                }
            } else if (c == '"') {
                in_double_quote = false;
            } else if (c == '\\') {
                escape_next = true;
            } else {
                try item.append(c);
            }
        } else if (escape_next) {
            escape_next = false;
            try item.append(c);
        } else {
            switch (c) {
                ' ' => {
                    if (item.items.len > 0) {
                        if (reOpt) |opt| {
                            var op = opt;
                            op.path = try item.toOwnedSlice();
                            try reList.append(op);
                            reOpt = null;
                            continue;
                        }
                        const slice = try item.toOwnedSlice();
                        reOpt = is_redirect(slice);
                        if (reOpt) |_| {
                            continue;
                        }
                        try result.append(slice);
                    }
                },
                '\'' => in_single_quote = true,
                '"' => in_double_quote = true,
                '\\' => escape_next = true,
                else => try item.append(c),
            }
        }
    }
    if (in_double_quote or in_single_quote) {
        return error.UnCLosedQuote;
    }
    if (item.items.len > 0) {
        if (reOpt) |opt| {
            var op = opt;
            op.path = try item.toOwnedSlice();
            try reList.append(op);
            reOpt = null;
        } else {
            try result.append(try item.toOwnedSlice());
        }
    }
    return .{ .args = try result.toOwnedSlice(), .reList = try reList.toOwnedSlice() };
}

fn is_redirect(args: []u8) ?redirectOpt {
    if (std.mem.eql(u8, args, ">") or std.mem.eql(u8, args, "1>")) {
        return redirectOpt{ .isAppend = false, .isStderr = false, .path = "" };
    } else if (std.mem.eql(u8, args, ">>") or std.mem.eql(u8, args, "1>>")) {
        return redirectOpt{ .isAppend = true, .isStderr = false, .path = "" };
    } else if (std.mem.eql(u8, args, "2>")) {
        return redirectOpt{ .isAppend = false, .isStderr = true, .path = "" };
    } else if (std.mem.eql(u8, args, "2>>")) {
        return redirectOpt{ .isAppend = true, .isStderr = true, .path = "" };
    } else {
        return null;
    }
}

fn is_executable(path: []u8) ?[]u8 {
    const file_name = std.mem.concat(Allocator, u8, &[_][]const u8{ path, SUFFIX }) catch return null;
    const path_env_map = std.process.getEnvMap(Allocator) catch return null;
    const path_env = path_env_map.get("PATH").?;
    var paths = std.mem.splitAny(u8, path_env, SPLIT);
    while (paths.next()) |p| {
        const com_path = std.fs.path.join(Allocator, &[_][]const u8{ p, file_name }) catch continue;
        const fs = std.fs.cwd().openFile(com_path, .{}) catch {
            Allocator.free(com_path);
            continue;
        };
        if (builtin.os.tag == .windows) {
            fs.close();
            return com_path;
        }
        if (fs.mode() & 0o111 == 0) {
            Allocator.free(com_path);
            continue;
        }
        fs.close();
        return com_path;
    }
    return null;
}

fn out_put(val: anyerror![]u8, re_list: []redirectOpt) void {
    var writer = std.io.getStdOut().writer();
    var std_val: []const u8 = undefined;
    var flag: bool = undefined;
    if (val) |v| {
        std_val = v;
        flag = true;
    } else |err| {
        std_val = @errorName(err);
        flag = false;
    }
    if (re_list.len == 0) {
        writer.print("{s}", .{std_val}) catch unreachable;
        return;
    }
    var out_flag = false;
    for (re_list) |re| {
        if (re.isStderr != flag) {
            out_flag = true;
            re_to_file(re, std_val) catch |err| {
                writer.print("error: {any}\n", .{err}) catch unreachable;
            };
        } else {
            re_to_file(re, "") catch |err| {
                writer.print("error: {any}\n", .{err}) catch unreachable;
            };
        }
    }
    if (!out_flag) {
        writer.print("{s}", .{std_val}) catch unreachable;
    }
}

fn re_to_file(re: redirectOpt, val: []const u8) !void {
    var file: ?std.fs.File = null;
    if (std.fs.cwd().openFile(re.path, .{ .mode = .read_write })) |f| {
        file = f;
    } else |err| {
        if (err != error.FileNotFound) {
            return err;
        }
        file = try std.fs.cwd().createFile(re.path, .{ .read = true });
    }
    var fs = file.?;
    defer fs.close();
    if (re.isAppend) {
        try fs.seekFromEnd(0);
    }
    try fs.writeAll(val);
}

fn execution(path: []u8, args: [][]u8) ![]u8 {
    var exec_args = std.ArrayList([]u8).init(Allocator);
    try exec_args.append(path);
    try exec_args.appendSlice(args);
    defer exec_args.deinit();
    var child_process = std.process.Child.init(exec_args.items, Allocator);
    child_process.stderr_behavior = .Pipe;
    child_process.stdout_behavior = .Pipe;
    try child_process.spawn();
    const std_out: []u8 = forward(child_process.stdout) catch "";
    const std_err: []u8 = forward(child_process.stderr) catch "";
    _ = try child_process.wait();
    if (std_err.len == 0) {
        return std_out;
    }
    return error.Error;
}

pub fn forward(file: ?std.fs.File) ![]u8 {
    var source: std.fs.File = undefined;
    if (file) |f| {
        source = f;
    } else {
        return &[_]u8{};
    }
    var buffer: [1024]u8 = undefined;
    var target = std.ArrayList(u8).init(Allocator);
    var writer = target.writer();
    while (true) {
        const bytes_read = try source.read(&buffer);
        if (bytes_read > 0) {
            try writer.writeAll(buffer[0..bytes_read]);
        } else {
            break;
        }
    }
    defer target.deinit();
    return try target.toOwnedSlice();
}
