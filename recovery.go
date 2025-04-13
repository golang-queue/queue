package queue

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
)

const (
	dunno     = "???"
	centerDot = "·"
	dot       = "."
	slash     = "/"
)

// bufferPool is a pool of byte buffers that can be reused to reduce the number
// of allocations and improve performance. It uses sync.Pool to manage a pool
// of reusable *bytes.Buffer objects. When a buffer is requested from the pool,
// if one is available, it is returned; otherwise, a new buffer is created.
// When a buffer is no longer needed, it should be put back into the pool to be
// reused.
var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// stack captures and returns the current stack trace, skipping the specified number of frames.
// It retrieves the stack trace information, including the file name, line number, and function name,
// and formats it into a byte slice. The function uses a buffer pool to manage memory efficiently.
//
// Parameters:
//   - skip: The number of stack frames to skip before recording the trace.
//
// Returns:
//   - A byte slice containing the formatted stack trace.
func stack(skip int) []byte {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source retrieves the n-th line from the provided slice of byte slices,
// trims any leading and trailing whitespace, and returns it as a byte slice.
// If n is out of range, it returns a default "dunno" byte slice.
//
// Parameters:
//
//	lines - a slice of byte slices representing lines of text
//	n - the 1-based index of the line to retrieve
//
// Returns:
//
//	A byte slice containing the trimmed n-th line, or a default "dunno" byte slice if n is out of range.
func source(lines [][]byte, n int) []byte {
	n--
	if n < 0 || n >= len(lines) {
		return []byte(dunno)
	}
	return bytes.TrimSpace(lines[n])
}

// function takes a program counter (pc) value and returns the name of the function
// corresponding to that program counter as a byte slice. It uses runtime.FuncForPC
// to retrieve the function information and processes the function name to remove
// any path and package information, returning only the base function name.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return []byte(dunno)
	}
	name := fn.Name()
	if lastSlash := strings.LastIndex(name, slash); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := strings.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = strings.ReplaceAll(name, centerDot, dot)
	return []byte(name)
}
