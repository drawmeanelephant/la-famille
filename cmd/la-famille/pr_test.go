package main

import "testing"

func TestPRSyncBaseBranchDefault(t *testing.T) {
	baseFlag := prSyncCmd.Flags().Lookup("base")
	if baseFlag == nil {
		t.Fatal("pr sync command is missing the base flag")
	}

	if got, want := baseFlag.DefValue, "master"; got != want {
		t.Errorf("base flag default = %q, want %q", got, want)
	}
}
