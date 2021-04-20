package model

import (
	"os"
	"testing"
)

func TestGetCurrentPid(t *testing.T) {
	tree, err := NewProcessTree()
	if err != nil {
		t.Errorf("Got error %v", err)
	}
	ppid := os.Getppid()
	pid := os.Getpid()
	childs := tree.Get(ppid)
	if !ContainsIntInIntSlice(childs, pid) {
		t.Errorf("Test wants pid %d in %v for ppid %d", pid, childs, ppid)
	}
}

func TestNonExistingPid(t *testing.T) {
	tree, err := NewProcessTree()
	if err != nil {
		t.Errorf("Got error %v", err)
	}
	pid := 32769
	childs := tree.Get(pid)
	if len(childs) > 0 {
		t.Errorf("want pid %d has no children %v", pid, childs)
	}
}
