package a_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestSkip(t *testing.T) {
	t.Skip("skip")
	t.Fatal("ShouldNotReach")
}

func TestShort(t *testing.T) {
	if testing.Short() {
		t.Logf("testing.Short=true")
		t.Skip()
	}
	t.Logf("testing.Short=false")
}

func TestCli(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Run("short option", func(t *testing.T) {
		output := bytes.NewBuffer(nil)
		cmd := exec.Command("go", "test", "github.kakaoenterprise.in/cloud-platform/go-learning/a-test", "-v", "-short", "-run", "TestShort")
		cmd.Stdout = output

		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), "testing.Short=true") {
			t.Error("expect testing.Short is true, actual is false")
		}
	})
	t.Run("no short option", func(t *testing.T) {
		output := bytes.NewBuffer(nil)
		cmd := exec.Command("go", "test", "github.kakaoenterprise.in/cloud-platform/go-learning/a-test", "-v", "-run", "TestShort")
		cmd.Stdout = output

		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), "testing.Short=false") {
			t.Error("expect testing.Short is false, actual is true")
		}
	})
}
