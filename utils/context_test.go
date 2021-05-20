package utils

import (
	"context"
	"testing"
	"time"
)

func TestRequestTimeoutFromContext(t *testing.T) {
	cases := []struct {
		got  time.Duration
		want interface{}
		ok   bool
	}{
		{
			got:  10 * time.Second,
			want: 10 * time.Second,
			ok:   true,
		},
		{
			got:  DefaultRequestTimeout,
			want: "10 * time.Second",
			ok:   false,
		},
	}
	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), CtxRequestTimeoutKey, tc.want)
		val, ok := RequestTimeoutFromContext(ctx)
		if ok != tc.ok {
			t.Errorf("ok %v != tc.ok %v ", ok, tc.ok)
		}
		if val != tc.got {
			t.Errorf("val %v != tc.got %v ", val, tc.got)
		}
	}
}

func TestFromRequestTimeout(t *testing.T) {
	want := 10 * time.Second
	ctx := FromRequestTimeout(context.Background(), want)
	val, ok := RequestTimeoutFromContext(ctx)
	if !ok {
		t.Errorf("Expecting true from RequestTimeoutFromContext")
	}
	if val != want {
		t.Errorf("val %v != want %v ", val, want)
	}

}
