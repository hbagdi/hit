package db

import (
	"testing"

	"github.com/hbagdi/hit/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	type args struct {
		opts StoreOpts
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "db without logger",
			args: args{
				opts: StoreOpts{
					Logger: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "db with logger",
			args: args{
				opts: StoreOpts{
					Logger: log.Logger,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStore(tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestStoreClose(t *testing.T) {
	store, err := NewStore(StoreOpts{Logger: log.Logger})
	require.NoError(t, err)
	require.NoError(t, store.Close())
}
