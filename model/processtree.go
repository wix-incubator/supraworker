package model

import (
	"fmt"
	ps "github.com/mitchellh/go-ps"
)

// Tree is a tree of processes.
type Tree struct {
	Procs map[int]Process
}

// Get all children by pid
func (t *Tree) Get(pid int) []int {
	if _, ok := t.Procs[pid]; !ok {
		return []int{}
	}
	res := t.Procs[pid].Children
	for _, cid := range t.Procs[pid].Children {
		res = append(res, t.Get(cid)...)
	}
	return res
}

// Process stores information about a UNIX process.
type Process struct {
	Stat     ps.Process
	Children []int
}

// ContainsIntInIntSlice check whether int in the slice
func ContainsIntInIntSlice(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Returns new Processes Tree
func NewProcessTree() (*Tree, error) {
	var treeProcessList map[int]Process
	processList, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	treeProcessList = make(map[int]Process, len(processList))

	for aux := range processList {
		process := processList[aux]
		proc := Process{Stat: process, Children: make([]int, 0)}
		treeProcessList[process.Pid()] = proc
	}

	for pid, proc := range treeProcessList {
		if proc.Stat.PPid() == 0 {
			continue
		}
		parent, ok := treeProcessList[proc.Stat.PPid()]
		if !ok {
			return nil, fmt.Errorf("parent pid=%d of pid=%d does not exist",
				proc.Stat.PPid(), pid,
			)
		}
		parent.Children = append(parent.Children, pid)
		treeProcessList[parent.Stat.Pid()] = parent
	}
	tree := &Tree{
		Procs: treeProcessList,
	}
	return tree, nil
}
