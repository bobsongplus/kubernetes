/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package keepalived

import (
	"testing"
)

func TestGenerateKeepalivedConfig(t *testing.T) {
	type args struct {
		lbs            []string
		vip            string
		keepalivedPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "keepalived",
			args: args{
				lbs: []string{
					"192.168.1.1",
					"192.168.2.1",
				},
				vip:            "192.168.1.18",
				keepalivedPath: "./keepalived.conf",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GenerateKeepalivedConfig(tt.args.lbs, tt.args.vip, tt.args.keepalivedPath); (err != nil) != tt.wantErr {
				t.Errorf("GenerateKeepalivedConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
