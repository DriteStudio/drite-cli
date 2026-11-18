package drite

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SSHReadinessStatus string

const (
	SSHReady       SSHReadinessStatus = "ready"
	SSHStarting    SSHReadinessStatus = "starting"
	SSHStopped     SSHReadinessStatus = "stopped"
	SSHUnsupported SSHReadinessStatus = "unsupported"
	SSHUnknown     SSHReadinessStatus = "unknown"
	SSHFailed      SSHReadinessStatus = "failed"
)

type SSHReadiness struct {
	Status    SSHReadinessStatus `json:"status"`
	Username  *string            `json:"username"`
	IP        *string            `json:"ip"`
	CheckedAt *string            `json:"checkedAt"`
}

var linuxSSHUsername = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// Target returns the validated Linux username and numeric IP of a ready guest.
func (r SSHReadiness) Target() (string, netip.Addr, error) {
	if r.Status != SSHReady || r.Username == nil || r.IP == nil {
		return "", netip.Addr{}, fmt.Errorf("drite: SSH readiness has no ready target")
	}
	if *r.Username == "root" || !linuxSSHUsername.MatchString(*r.Username) {
		return "", netip.Addr{}, fmt.Errorf("drite: SSH readiness returned an invalid Linux username")
	}
	ip, err := netip.ParseAddr(*r.IP)
	if err != nil || ip.Zone() != "" || ip.IsUnspecified() {
		return "", netip.Addr{}, fmt.Errorf("drite: SSH readiness returned an invalid IP address")
	}
	return *r.Username, ip, nil
}

type WaitForSSHOptions struct {
	// Timeout defaults to five minutes. An earlier context deadline takes precedence.
	Timeout time.Duration
	// Interval defaults to two seconds and accepts values from one to thirty seconds.
	Interval time.Duration
}

func (s *VPSService) SSHReadiness(ctx context.Context, vpsID string) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path+"/ssh-readiness", nil, nil)
}

// WaitForSSH polls while readiness is pending or the API returns HTTP 429 or 503.
// Ready confirms that the guest has set up SSH. Reachability and client
// authentication still depend on the network, firewall, and credentials.
func (s *VPSService) WaitForSSH(ctx context.Context, vpsID string, options WaitForSSHOptions) (*Response, error) {
	timeout, interval := options.Timeout, options.Interval
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	if timeout < 0 {
		return nil, fmt.Errorf("drite: SSH wait timeout must be positive")
	}
	if interval == 0 {
		interval = 2 * time.Second
	}
	if interval < time.Second || interval > 30*time.Second {
		return nil, fmt.Errorf("drite: SSH polling interval must be between 1s and 30s")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var lastStatus string
	for {
		response, err := s.SSHReadiness(ctx, vpsID)
		delay := interval
		if err != nil {
			if ctx.Err() != nil {
				return nil, sshWaitContextError(vpsID, lastStatus, ctx.Err())
			}
			if response == nil || (response.StatusCode != http.StatusServiceUnavailable &&
				response.StatusCode != http.StatusTooManyRequests) {
				return response, err
			}
			lastStatus = fmt.Sprintf("HTTP %d", response.StatusCode)
			delay = sshPollDelay(interval, response.Header.Get("Retry-After"))
		} else {
			var envelope struct {
				Data SSHReadiness `json:"data"`
			}
			if err := response.Decode(&envelope); err != nil {
				return nil, err
			}
			status := envelope.Data.Status
			lastStatus = string(status)
			switch status {
			case SSHReady:
				if _, _, err := envelope.Data.Target(); err != nil {
					return nil, err
				}
				return response, nil
			case SSHStopped, SSHUnsupported, SSHFailed:
				return nil, fmt.Errorf("drite: VPS %q cannot become SSH ready (status %s)", vpsID, status)
			case SSHStarting, SSHUnknown:
			default:
				return nil, fmt.Errorf("drite: VPS %q returned an invalid SSH readiness status", vpsID)
			}
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, sshWaitContextError(vpsID, lastStatus, ctx.Err())
		case <-timer.C:
		}
	}
}

func sshPollDelay(interval time.Duration, retryAfter string) time.Duration {
	retryAfter = strings.TrimSpace(retryAfter)
	if seconds, err := strconv.ParseUint(retryAfter, 10, 64); err == nil {
		const maxSeconds = (1<<63 - 1) / uint64(time.Second)
		return max(interval, time.Duration(min(seconds, maxSeconds))*time.Second)
	}
	if at, err := http.ParseTime(retryAfter); err == nil {
		return max(interval, time.Until(at))
	}
	return interval
}

func sshWaitContextError(vpsID, status string, err error) error {
	if status == "" {
		return fmt.Errorf("drite: waiting for SSH on VPS %q: %w", vpsID, err)
	}
	return fmt.Errorf("drite: waiting for SSH on VPS %q (last status %s): %w", vpsID, status, err)
}
