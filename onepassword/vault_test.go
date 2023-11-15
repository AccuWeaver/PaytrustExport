package onepassword

import (
	"reflect"
	"testing"
)

func TestGetMFAToken(t *testing.T) {
	type args struct {
		alias        string
		vaultEntries VaultEntries
	}
	tests := []struct {
		name               string
		args               args
		wantTokenAccountId string
		wantToken          string
		wantErr            bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTokenAccountId, gotToken, err := GetMFAToken(tt.args.alias, tt.args.vaultEntries)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMFAToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTokenAccountId != tt.wantTokenAccountId {
				t.Errorf("GetMFAToken() gotTokenAccountId = %v, want %v", gotTokenAccountId, tt.wantTokenAccountId)
			}
			if gotToken != tt.wantToken {
				t.Errorf("GetMFAToken() gotToken = %v, want %v", gotToken, tt.wantToken)
			}
		})
	}
}

func TestGetPassword(t *testing.T) {
	type args struct {
		alias        string
		vaultEntries VaultEntries
	}
	tests := []struct {
		name         string
		args         args
		wantPassword string
		wantErr      bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPassword, err := GetPassword(tt.args.alias, tt.args.vaultEntries)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPassword != tt.wantPassword {
				t.Errorf("GetPassword() gotPassword = %v, want %v", gotPassword, tt.wantPassword)
			}
		})
	}
}

func TestGetVaultEntries(t *testing.T) {
	tests := []struct {
		name             string
		wantVaultEntries VaultEntries
	}{
		// TODO: Add test cases.
		{
			name:             "Test me",
			wantVaultEntries: VaultEntries{VaultEntry{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotVaultEntries := GetVaultEntries("Adobe", ""); !reflect.DeepEqual(gotVaultEntries, tt.wantVaultEntries) {
				t.Errorf("GetVaultEntries() = %#v, want %#v", gotVaultEntries, tt.wantVaultEntries)
			}
		})
	}
}

func TestGetVaultEntry(t *testing.T) {
	type args struct {
		alias        string
		vaultEntries VaultEntries
	}
	tests := []struct {
		name           string
		args           args
		wantVaultEntry VaultEntry
		wantFound      bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVaultEntry, gotFound := AddToVaultIfNotFound(tt.args.alias, tt.args.vaultEntries)
			if !reflect.DeepEqual(gotVaultEntry, tt.wantVaultEntry) {
				t.Errorf("AddToVaultIfNotFound() gotVaultEntry = %v, want %v", gotVaultEntry, tt.wantVaultEntry)
			}
			if gotFound != tt.wantFound {
				t.Errorf("AddToVaultIfNotFound() gotFound = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestVaultEntryHasMFA(t *testing.T) {
	type args struct {
		entry VaultEntry
	}
	tests := []struct {
		name       string
		args       args
		wantHasMFA bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotHasMFA := VaultEntryHasMFA(tt.args.entry); gotHasMFA != tt.wantHasMFA {
				t.Errorf("VaultEntryHasMFA() = %v, want %v", gotHasMFA, tt.wantHasMFA)
			}
		})
	}
}
