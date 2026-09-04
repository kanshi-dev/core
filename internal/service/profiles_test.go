package service

import (
	"errors"
	"testing"
)

func TestProfileCaptureInputValidation(t *testing.T) {
	valid := []ProfileCaptureInput{
		{TargetName: "checkout", ProfileType: "cpu", DurationSeconds: 5},
		{TargetName: "checkout", ProfileType: "cpu", DurationSeconds: 10},
		{TargetName: "checkout", ProfileType: "cpu", DurationSeconds: 30},
		{TargetName: "checkout", ProfileType: "trace", DurationSeconds: 1},
		{TargetName: "checkout", ProfileType: "trace", DurationSeconds: 5},
		{TargetName: "checkout", ProfileType: "heap"},
		{TargetName: "checkout", ProfileType: "allocs"},
		{TargetName: "checkout", ProfileType: "goroutine"},
		{TargetName: "checkout", ProfileType: "mutex"},
		{TargetName: "checkout", ProfileType: "block"},
		{TargetName: "checkout", ProfileType: "threadcreate"},
	}
	for _, in := range valid {
		if err := in.validate(); err != nil {
			t.Errorf("%+v: %v", in, err)
		}
	}
	for _, in := range []ProfileCaptureInput{
		{TargetName: "checkout", ProfileType: "cpu", DurationSeconds: 1},
		{TargetName: "checkout", ProfileType: "trace", DurationSeconds: 30},
		{TargetName: "checkout", ProfileType: "heap", DurationSeconds: 1},
		{TargetName: "checkout", ProfileType: "unknown"},
		{ProfileType: "heap"},
	} {
		if err := in.validate(); !errors.Is(err, ErrInvalidProfile) {
			t.Errorf("%+v: got %v", in, err)
		}
	}
}
