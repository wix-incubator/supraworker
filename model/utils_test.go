package model

import (
	"fmt"
	"go.uber.org/goleak"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestDefaultPath(t *testing.T) {
	path := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", path)
	}()
	os.Unsetenv("PATH")

	got := DefaultPath()
	if len(got) < 1 {
		t.Errorf("want len > 0 got %s", got)
	}
	want := "PATH="
	if !strings.Contains(got, want) {
		t.Errorf("want %s, got %v", want, got)
	}
}

func TestMergeEnvVars(t *testing.T) {
	path := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", path)
	}()
	os.Unsetenv("PATH")
	os.Unsetenv("SUPRAWORKER_T")
	want := "SUPRAWORKER_T=test"
	got := MergeEnvVars([]string{want})
	if len(got) < 1 {
		t.Errorf("want len > 0 got %s", got)
	}
	joined := strings.Join(got, ";")
	if !strings.Contains(joined, want) {
		t.Errorf("want %s, got %v", want, got)
	}
	want = "PATH="
	if !strings.Contains(joined, want) {
		t.Errorf("want %s, got %v", want, got)
	}
}

func TestUseCmdAsIs(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{
			cmd:  "adasasdasdasdasdasdasdasdasdasdasdasd asd",
			want: false,
		},
		{
			cmd:  "bash -c id",
			want: true,
		},
		{
			cmd:  "/bin/bash -c id",
			want: true,
		},
		{
			cmd:  "/bin/sh -c id",
			want: true,
		},
		{
			cmd:  "sh -c id",
			want: true,
		},
		// test if not shell but file exists
		{
			cmd:  fmt.Sprintf("%s arg id", os.Args[0]),
			want: false,
		},
	}

	for _, tc := range cases {
		got := useCmdAsIs(tc.cmd)
		if tc.want != got {
			t.Errorf("want %v, got %v", tc.want, got)
		}
	}
}

func TestCmdWrapper(t *testing.T) {
	cases := []struct {
		UseSHELL  bool
		CMD       string
		wantShell string
		wantArgs  []string
	}{
		{
			UseSHELL:  false,
			CMD:       "cmd",
			wantShell: "cmd",
			wantArgs:  []string{},
		},
		{
			UseSHELL:  false,
			CMD:       "cmd arg1",
			wantShell: "cmd",
			wantArgs:  []string{"arg1"},
		},
		{
			UseSHELL:  false,
			CMD:       "cmd arg1 --arg2",
			wantShell: "cmd",
			wantArgs:  []string{"arg1", "--arg2"},
		},

		{
			UseSHELL:  true,
			CMD:       "bash cmd",
			wantShell: "bash",
			wantArgs:  []string{"cmd"},
		},
		{
			UseSHELL:  true,
			CMD:       "bash cmd arg1",
			wantShell: "bash",
			wantArgs:  []string{"cmd", "arg1"},
		},
		{
			UseSHELL:  true,
			CMD:       "bash cmd arg1 --arg2",
			wantShell: "bash",
			wantArgs:  []string{"cmd", "arg1", "--arg2"},
		},
	}

	for i, tc := range cases {
		shell, args := CmdWrapper("", tc.UseSHELL, tc.CMD)
		if tc.wantShell != shell {
			t.Errorf("want %v, got %v", tc.wantShell, shell)
		}

		sort.Sort(sort.Reverse(sort.StringSlice(tc.wantArgs)))
		sort.Sort(sort.Reverse(sort.StringSlice(args)))
		if strings.Join(tc.wantArgs, ";") != strings.Join(args, ";") {
			t.Errorf("want %v, got %v", tc.wantArgs, args)
		}
		user := fmt.Sprintf("user%d", i)
		shell, args = CmdWrapper(user, tc.UseSHELL, tc.CMD)
		sort.Sort(sort.Reverse(sort.StringSlice(args)))

		if !strings.Contains(strings.Join(args, " "), strings.Join(tc.wantArgs, " ")) {
			t.Errorf("want %s, got %v", strings.Join(tc.wantArgs, " "), strings.Join(args, " "))
		}
		// check that we are using run as
		want := "su"
		switch runtime.GOOS {
		case "windows":
			want = "runas"
		}

		if !strings.Contains(shell, want) {
			t.Errorf("want %s, got %v", want, shell)
		}
		// check that user is added to command
		want = user
		if !strings.Contains(strings.Join(args, " "), want) {
			t.Errorf("want %s, got %v", want, strings.Join(args, " "))
		}

	}
}
