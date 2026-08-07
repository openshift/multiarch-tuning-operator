package operator

import (
	"errors"
	"testing"
	"time"
)

func TestRequeueAfterError_ErrorsAs(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		msg      string
	}{
		{
			name:     "5s requeue for progressing",
			duration: 5 * time.Second,
			msg:      clusterPodPlacementConfigNotReady,
		},
		{
			name:     "10s requeue for ungating",
			duration: 10 * time.Second,
			msg:      waitingForUngatingPodsError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newRequeueAfterError(tt.duration, tt.msg)
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			if err.Error() != tt.msg {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.msg)
			}

			var requeueErr *requeueAfterError
			if !errors.As(err, &requeueErr) {
				t.Fatal("errors.As should unwrap requeueAfterError")
			}
			if requeueErr.duration != tt.duration {
				t.Errorf("duration = %v, want %v", requeueErr.duration, tt.duration)
			}
			if requeueErr.msg != tt.msg {
				t.Errorf("msg = %q, want %q", requeueErr.msg, tt.msg)
			}
		})
	}
}

func TestRequeueAfterError_NotMatchPlainError(t *testing.T) {
	plainErr := errors.New("some error")
	var requeueErr *requeueAfterError
	if errors.As(plainErr, &requeueErr) {
		t.Fatal("errors.As should not match plain error as requeueAfterError")
	}
}
