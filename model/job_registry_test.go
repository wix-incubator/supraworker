package model

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkRegistryAdd(b *testing.B) {
	r := NewRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := NewJob(fmt.Sprintf("job-%v", b.N), "echo")
		r.Add(job)
	}
}

func BenchmarkRegistryCleanUp(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewRegistry()
		for ii := 0; ii < 100; ii++ {
			job := NewJob(fmt.Sprintf("job-%v", b.N), "echo")
			r.Add(job)
			r.Cleanup()
		}
	}
}

func TestRegistryAddNoDuplicateJob(t *testing.T) {
	r := NewRegistry()
	for ii := 0; ii < 100; ii++ {
		job := NewJob(fmt.Sprintf("job-%v", ii), "echo")
		if !r.Add(job) {
			t.Errorf("Expect to add job")
		}
		for j := 0; j < 10; j++ {
			if r.Add(job) {
				t.Errorf("Expect not to add job")

			}
		}

	}
}

func TestRegistryLen(t *testing.T) {
	r := NewRegistry()
	num := 100
	for ii := 0; ii < num; ii++ {
		job := NewJob(fmt.Sprintf("job-%v", ii), "echo")
		if !r.Add(job) {
			t.Errorf("Expect to add job")
		}
	}
	if r.Len() != num {
		t.Errorf("Expect %v got length %v", num, r.Len())
	}
}

func TestRegistryDelete(t *testing.T) {
	r := NewRegistry()
	num := 100
	for ii := 0; ii < num; ii++ {
		job := NewJob(fmt.Sprintf("job-%v", ii), "echo")
		if !r.Add(job) {
			t.Errorf("Expect to add job")
		}
		if !r.Delete(job.StoreKey()) {
			t.Errorf("Expect to delete job")
		}
		if r.Delete(job.StoreKey()) {
			t.Errorf("Expect the job to be already deleted")
		}

	}
	if r.Len() != 0 {
		t.Errorf("Expect %v got length %v", num, r.Len())
	}
}

func TestRegistryCleanup(t *testing.T) {
	r := NewRegistry()
	num := 100
	for ii := 0; ii < num; ii++ {
		job := NewJob(fmt.Sprintf("job-%v", ii), "echo")
		job.TTR = 10000
		// no cancelation flow on cleanup
		// right now it won't execute something
		job.Status = JOB_STATUS_CANCELED

		if !r.Add(job) {
			t.Errorf("Expect to add job")
		}
		n := r.Len()
		if (r.Cleanup() > 0) || (r.Len() != n) {
			t.Errorf("Expect no job to be already deleted by Cleanup")
		}
		job.StartAt = time.Now().Add(time.Duration(-10001) * time.Millisecond)
		if (r.Cleanup() == 0) || (r.Len() == n) {
			t.Errorf("Expect Job to be deleted by Cleanup due to TTR")
		}

	}
	if r.Len() != 0 {
		t.Errorf("Expect %v got length %v", num, r.Len())
	}
}
