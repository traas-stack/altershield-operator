package utils

import (
	"context"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

// ExponentialBackoffWithContext works similar like wait.ExponentialBackoffWithContext but with below differences:
// * It does not stop when the cap of backoff is reached.
// * It does not return the error of ctx when done.
func ExponentialBackoffWithContext(ctx context.Context, backoff wait.Backoff, condition wait.ConditionWithContextFunc) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if ok, err := runConditionWithCrashProtectionWithContext(ctx, condition); err != nil || ok {
			return err
		}

		t := time.NewTimer(backoff.Step())
		select {
		case <-ctx.Done():
			t.Stop()
			return nil
		case <-t.C:
		}
	}
}

// runConditionWithCrashProtectionWithContext is copied from k8s.io/apimachinery/pkg/util/wait.
func runConditionWithCrashProtectionWithContext(ctx context.Context, condition wait.ConditionWithContextFunc) (bool, error) {
	defer runtime.HandleCrash()
	return condition(ctx)
}
