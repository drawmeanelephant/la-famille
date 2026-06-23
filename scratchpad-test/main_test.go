package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"
)

func TestSomething(t *testing.T) {
	fmt.Println("Running go test")
	cmd := exec.Command("go", "test", "-v", "./...")
	cmd.Dir = "../"
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	fmt.Println(out.String())
	if err != nil {
		t.Fatalf("go test failed: %v", err)
	}
}
