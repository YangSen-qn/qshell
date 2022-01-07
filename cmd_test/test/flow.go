package test

import (
	"fmt"
	"os/exec"
)

type testFlow struct {
	args          []string
	resultHandler func(line string)
	errorHandler  func(line string)
}

func NewTestFlow(args ...string) *testFlow {
	return &testFlow{
		args: args,
	}
}

func (t *testFlow) ResultHandler(handler func(line string)) *testFlow {
	t.resultHandler = handler
	return t
}

func (t *testFlow) ErrorHandler(handler func(line string)) *testFlow {
	t.errorHandler = handler
	return t
}

func (t *testFlow) Run() {
	fmt.Println("")
	fmt.Println("========== CMD Start:", t.args, "==========")

	cmd := exec.Command("qshell", t.args...)
	cmd.Stdout = newLineWriter(t.resultHandler)
	cmd.Stderr = newLineWriter(t.errorHandler)

	if err := cmd.Run(); err != nil {
		fmt.Println("========== CMD   End:", t.args, " error:", err.Error(), "==========")
	} else {
		fmt.Println("========== CMD   End:", t.args, "==========")
	}

	fmt.Println("")
}