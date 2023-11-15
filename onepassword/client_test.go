package onepassword

import (
	"log"
	"testing"
)

func TestClient_runJson(t *testing.T) {
	type fields struct {
		args []string
		path string
	}
	type args struct {
		data any
		args []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				Args: tt.fields.args,
				path: tt.fields.path,
			}
			if err := c.GetJSON(tt.args.data, tt.args.args...); (err != nil) != tt.wantErr {
				t.Errorf("GetJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_runPlain(t *testing.T) {
	type fields struct {
		args []string
		path string
	}
	type args struct {
		args []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				Args: tt.fields.args,
				path: tt.fields.path,
			}
			got, err := c.RunPlain(tt.args.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunPlain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RunPlain() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	type args struct {
		opts *ClientOptions
	}
	tests := []struct {
		name    string
		args    args
		want    *Client
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "Simple test",
			args: args{opts: &ClientOptions{
				Account: nil,
				Cache:   nil,
				Config:  nil,
				Session: nil,
			}},
			want: &Client{
				Args: []string{
					"op", "--format", "json", "--iso-timestamps", "--no-color", "--tags Workspace", "--account", "Adobe",
				},
				path: "/usr/local/bin/op",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("NewClient() got = %#v, want %#v", got, tt.want)
			//}
			var vaultEntries []VaultEntry
			err = got.GetJSON(&vaultEntries, "item", "list", "--tags", "Workspace")
			if err != nil {
				t.Errorf("GetJSON error = %v", err)
				return
			}

			log.Printf("%d found", len(vaultEntries))
			if len(vaultEntries) != 25 {
				t.Errorf("got %d entries, want 25", len(vaultEntries))
				return
			}
		})
	}
}
