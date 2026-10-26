// Copyright 2025 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ats

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// checkNumGoroutines waits up to 2 seconds for the number of go routines to
// satisfy the given condition. Goroutines take a moment to start or exit, so
// a single count is racy.
func checkNumGoroutines(t *testing.T, cond func(int) bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if n := runtime.NumGoroutine(); cond(n) {
			return
		} else if time.Now().After(deadline) {
			t.Fatalf("Expected %s, got %d go routines", msg, n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestNotRunningValue(t *testing.T) {
	// Set back to zero in case this test gets run multiple times via --count.
	utime.Store(0)
	at := AccessTime()
	if at == 0 {
		t.Fatal("Expected non-zero access time")
	}

	atn := AccessTime()
	if atn != at {
		t.Fatal("Did not expect updates to access time")
	}
}

func TestRegisterAndUnregister(t *testing.T) {
	ngrp := runtime.NumGoroutine()

	Register()
	if r := refs.Load(); r != 1 {
		t.Fatalf("Expected refs to be 1, got %d", r)
	}
	// Make sure we have a non-zero time.
	at := AccessTime()
	if at == 0 {
		t.Fatal("Expected non-zero access time")
	}
	// Make sure we are updating access time.
	time.Sleep(2 * TickInterval)
	atn := AccessTime()
	if atn == at {
		t.Fatal("Expected access time to be updated but it was not")
	}

	// Now unregister.
	Unregister()
	if r := refs.Load(); r != 0 {
		t.Fatalf("Expected refs to be 0, got %d", r)
	}
	time.Sleep(TickInterval)

	at = AccessTime()
	time.Sleep(2 * TickInterval)
	atn = AccessTime()
	if atn != at {
		t.Fatal("Did not expect updates to access time")
	}

	// Check that we have no additional go routines running. The ATS go routine
	// may need a moment to observe the ref count going to zero, and the baseline
	// may have been taken while a go routine from a previous run was exiting, so
	// only assert that we are not above it.
	checkNumGoroutines(t, func(n int) bool { return n <= ngrp },
		fmt.Sprintf("no additional go routines after removing all registered (baseline %d)", ngrp))
	ngra := runtime.NumGoroutine()

	// Check that we spin back up on going from zero to one registered after spinning down.
	Register()
	defer Unregister()

	at = AccessTime()
	time.Sleep(2 * TickInterval)
	atn = AccessTime()
	if atn == at {
		t.Fatal("Expected access time to be updated but it was not")
	}
	checkNumGoroutines(t, func(n int) bool { return n == ngra+1 },
		fmt.Sprintf("additional go routine after registering (was %d)", ngra))
}

func TestUnbalancedUnregister(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected function to panic, but it did not")
		}
	}()
	Unregister()
}
