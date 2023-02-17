package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddLink(t *testing.T) {
	type args struct {
		gToken  string
		longURL string
	}
	tests := []struct {
		name    string
		s       StorageLinks
		args    args
		wantErr bool
	}{
		{
			name: "Positive test",
			s: StorageLinks{
				LinksMap: map[string]string{},
			},
			args: args{
				gToken:  "AsDfGhJkLl",
				longURL: "https://www.youtube.com/",
			},
			wantErr: false,
		},
		{
			name: "Negative test test",
			s: StorageLinks{
				LinksMap: map[string]string{
					"AsDfGhJkLl": "https://go.dev/",
				},
			},
			args: args{
				gToken:  "AsDfGhJkLl",
				longURL: "https://www.youtube.com/",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.AddLink(tt.args.gToken, tt.args.longURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("StorageLinks.GetLongURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetLongURL(t *testing.T) {
	type args struct {
		sToken string
	}
	tests := []struct {
		name    string
		s       StorageLinks
		args    args
		wantURL string
		wantErr bool
	}{
		{
			name: "Positive test",
			s: StorageLinks{
				LinksMap: map[string]string{
					"AsDfGhJkLl": "https://go.dev/",
				},
			},
			args: args{
				sToken: "AsDfGhJkLl",
			},
			wantURL: "https://go.dev/",
			wantErr: false,
		},
		{
			name: "Negative test test",
			s: StorageLinks{
				LinksMap: map[string]string{
					"AsDfGhJkLl": "https://go.dev/",
				},
			},
			args: args{
				sToken: "7EJUYUVAMy",
			},
			wantURL: "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetLongURL(tt.args.sToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("StorageLinks.GetLongURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, got, tt.wantURL)
		})
	}
}
