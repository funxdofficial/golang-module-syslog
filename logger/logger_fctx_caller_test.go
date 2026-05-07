package logger

import (
	"strings"
	"testing"
)

// mirrorEnqueueFormattedCtxStack meniru rantai pemanggil *fCtx -> enqueueFormattedCtx -> getCallerInfo(3).
func mirrorEnqueueFormattedCtxStack() (file string, line int, fn string) {
	return getCallerInfo(3)
}

func wrapperCallsMirrorLikeSuccessfCtx() (file string, line int, fn string) {
	return mirrorEnqueueFormattedCtxStack()
}

// TestFormattedCtxCallerStackDepth memastikan getCallerInfo(3) dari helper dalam modul
// mengarah ke pemanggil aplikasi (file tes), bukan logger.go.
func TestFormattedCtxCallerStackDepth(t *testing.T) {
	file, line, fn := wrapperCallsMirrorLikeSuccessfCtx()
	if !strings.HasSuffix(file, "logger_fctx_caller_test.go") {
		t.Errorf("caller file = %q, want suffix logger_fctx_caller_test.go", file)
	}
	if line <= 0 {
		t.Errorf("caller line = %d", line)
	}
	if fn == "" {
		t.Errorf("caller function empty")
	}
	if fn == "mirrorEnqueueFormattedCtxStack" || fn == "wrapperCallsMirrorLikeSuccessfCtx" {
		t.Errorf("caller function leaked inner helper: %q", fn)
	}
}
