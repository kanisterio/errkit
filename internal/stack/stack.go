package stack

import (
	"runtime"
	"strings"
)

func GetLocationFromStack(stack []uintptr, callers int) (function, file string, line int) {
	if callers < 1 {
		// Failure potentially due to wrongly specified depth
		return "Unknown", "Unknown", 0
	}

	frames := runtime.CallersFrames(stack[:callers])
	var frame runtime.Frame
	frame, _ = frames.Next()
	filename := frame.File
	if paths := strings.SplitAfterN(frame.File, "/go/src/", 2); len(paths) > 1 {
		filename = paths[1]
	}

	return frame.Function, filename, frame.Line
}
