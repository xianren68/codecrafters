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
// const SPLIT = comptime if (std.builtin.os.tag == .linux) ":" else ";";
// const SUFFIX = comptime if (std.builtin.os.tag == .linux) ".exe" else "";
const HOME_ENV = if (builtin.os.tag == .windows) "USERPROFILE" else "HOME";
pub fn main() !void {
    handleMap = hashMap.init(std.heap.page_allocator);
    defer handleMap.deinit();
    try handleMap.put("exit", handle_exit);
    try handleMap.put("echo", handle_echo);
    try handleMap.put("type", handle_type);
    try handleMap.put("pwd", handle_pwd);
    try handleMap.put("cd", handle_cd);
    var writer = std.io.getStdOut().writer();
    var reader = std.io.getStdIn().reader();
    while (true) {
        try writer.print("$ ", .{});
        var buffer: [1024]u8 = undefined;
        const input = try reader.readUntilDelimiter(&buffer, '\n');
        if (input.len == 0) {
            continue;
        }
        if (parse_args(input[0 .. input.len - 1])) |result| {
            const args = result.args;
            _ = result.reList;
            if (args.len == 0) {
                continue;
            }
            const cmd = args[0];
            if (handleMap.get(cmd)) |handlerFn| {
                const res = try handlerFn(args[1..]);
                try writer.print("{s}", .{res});
            } else {
                try writer.print("{s}\n", .{input});
            }
        } else |err| {
            try writer.print("error: {any}\n", .{err});
            continue;
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
