/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package haproxy

import "testing"

func TestCreateHaproxyConfig(t *testing.T) {
	type args struct {
		lbs         []string
		haproxy_cfg string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				lbs:         []string{"192.168.1.21", "192.168.2.31"},
				haproxy_cfg: "./haproxy.cnf",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateHaproxyConfig(tt.args.lbs, tt.args.haproxy_cfg); (err != nil) != tt.wantErr {
				t.Errorf("CreateHaproxyConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateLocalHaproxyStaticPodManifestFile(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateLocalHaproxyStaticPodManifestFile(); (err != nil) != tt.wantErr {
				t.Errorf("CreateLocalHaproxyStaticPodManifestFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
