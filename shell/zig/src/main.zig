const std = @import("std");

const Allocator = std.heap.page_allocator;
const handler = fn ([][]u8) anyerror![]u8;
const hashMap = std.StringHashMap(*const handler);
var handleMap: hashMap = undefined;
// const SPLIT = comptime if (std.builtin.os.tag == .linux) ":" else ";";
// const SUFFIX = comptime if (std.builtin.os.tag == .linux) ".exe" else "";
pub fn main() !void {
    handleMap = hashMap.init(std.heap.page_allocator);
    defer handleMap.deinit();
    var writer = std.io.getStdOut().writer();
    var reader = std.io.getStdIn().reader();
    while (true) {
        try writer.print("$ ", .{});
        var buffer: [1024]u8 = undefined;
        const input = try reader.readUntilDelimiter(&buffer, '\n');
        if (parse_args(input)) |args| {
            try writer.print("error: {any}\n", .{args});
            continue;
        } else |err| {
            try writer.print("error: {any}\n", .{err});
            continue;
        }
        try writer.print("{s}\n", .{input});
    }
}

fn parse_args(args: []u8) ![][]u8 {
    var result = try std.ArrayList([]u8).initCapacity(Allocator, 200);
    defer result.deinit();
    var item = try std.ArrayList(u8).initCapacity(Allocator, 200);
    defer item.deinit();
    var in_single_quote = false;
    var in_double_quote = false;
    var escape_next = false;
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
                        try result.append(try item.toOwnedSlice());
                        item.clearAndFree();
                    }
                },
                '\'' => in_single_quote = true,
                '"' => in_double_quote = true,
                '\\' => escape_next = true,
                else => try item.append(c),
            }
        }
        if (in_double_quote or in_single_quote) {
            return error.UnCLosedQuote;
        }
        if (item.items.len > 0) {
            try result.append(try item.toOwnedSlice());
        }
    }
    return try result.toOwnedSlice();
}
