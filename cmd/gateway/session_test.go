package main

import (
	"testing"

	"github.com/clusterit/orca/config"
)

func TestCheckBackendAccessAllowDeny(t *testing.T) {
	allAllowed := config.Gateway{
		AllowDeny:    true,
		AllowedCidrs: []string{"0.0.0.0/0"},
	}

	ips := []string{"1.2.3.4", "192.168.0.4", "2.4.5.6"}
	for _, ip := range ips {
		err := checkBackendAccess(ip, allAllowed)
		if err != nil {
			t.Errorf("%s should be allowed", ip)
		}
	}
	allExceptAllowed := config.Gateway{
		AllowDeny:    true,
		AllowedCidrs: []string{"0.0.0.0/0"},
		DeniedCidrs:  []string{"2.4.5.6/24"},
	}

	for _, ip := range ips {
		err := checkBackendAccess(ip, allExceptAllowed)
		if ip == "2.4.5.6" && err == nil {
			t.Errorf("%s should not be allowed", ip)
		}
	}

	oneAllowed := config.Gateway{
		AllowDeny:    true,
		AllowedCidrs: []string{"192.168.0.1/24"},
	}
	for _, ip := range ips {
		err := checkBackendAccess(ip, oneAllowed)
		if ip == "192.168.0.4" {
			if err != nil {
				t.Errorf("%s should be allowed", ip)
			}
		} else {
			if err == nil {
				t.Errorf("%s should be denied", ip)
			}
		}
	}
}

func TestCheckBackendAccessDenyAllow(t *testing.T) {
	allDenied := config.Gateway{
		AllowDeny:   false,
		DeniedCidrs: []string{"0.0.0.0/0"},
	}

	ips := []string{"1.2.3.4", "192.168.0.4", "2.4.5.6"}
	for _, ip := range ips {
		err := checkBackendAccess(ip, allDenied)
		if err == nil {
			t.Errorf("%s should not be allowed", ip)
		}
	}
	oneAllowed := config.Gateway{
		AllowDeny:    false,
		AllowedCidrs: []string{"192.168.0.1/24"},
		DeniedCidrs:  []string{"0.0.0.0/0"},
	}
	for _, ip := range ips {
		err := checkBackendAccess(ip, oneAllowed)
		if ip == "192.168.0.4" {
			if err != nil {
				t.Errorf("%s should be allowed", ip)
			}
		} else {
			if err == nil {
				t.Errorf("%s should be denied", ip)
			}
		}
	}
	oneDenied := config.Gateway{
		AllowDeny:   false,
		DeniedCidrs: []string{"192.168.0.1/24"},
	}
	for _, ip := range ips {
		err := checkBackendAccess(ip, oneDenied)
		if ip == "192.168.0.4" {
			if err == nil {
				t.Errorf("%s should be denied", ip)
			}
		} else {
			if err != nil {
				t.Errorf("%s should be allowed", ip)
			}
		}
	}
}
