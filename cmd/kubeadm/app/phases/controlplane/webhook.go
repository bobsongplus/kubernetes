/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package controlplane

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/uuid"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

//func Hook(tceAddress, tceCredential, clusterId, k8sAddress, k8sToken string) error {
func Hook(cfg *kubeadmapi.InitConfiguration) error {
	credential := strings.Split(cfg.ApiServerCredential, ":")
	if len(credential) < 2 {
		return fmt.Errorf("[webhook] There's not valid cridiential for TenxCloud Enterprise Server, please provide correct one \n")
	}
	params := make(map[string]string)
	id := randInt(0, 1000)
	const CLUSTER_NAME_PREFIX = "k8s-"
	params["name"] = CLUSTER_NAME_PREFIX + strconv.Itoa(id)
	if "" != cfg.ClusterName {
		params["cluster_id"] = cfg.ClusterName
	}
	// Generate ControlPlane Enpoint kubeconfig file
	controlPlaneEndpoint, err := kubeadmutil.GetControlPlaneEndpoint(cfg.ControlPlaneEndpoint, &cfg.LocalAPIEndpoint)
	if err != nil {
		return err
	}
	params["access_url"] = controlPlaneEndpoint
	params["token"] = cfg.BootstrapTokens[0].Token.Secret
	params["description"] = CLUSTER_NAME_PREFIX + strconv.Itoa(id)
	playload, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("[webhook] Failed to consturct an HTTP request [%v] \n", err)
	}
	req, err := http.NewRequest("POST", cfg.ApiServerUrl, bytes.NewBuffer(playload))
	if err != nil {
		return fmt.Errorf("[webhook] Failed to consturct an HTTP request [%v] \n", err)
	}
	req.Header.Set("username", credential[0])
	req.Header.Set("authorization", fmt.Sprintf("token %s", credential[1]))
	//if the url is https, will skip the verify
	client := &http.Client{}
	if uri, err := url.Parse(cfg.ApiServerUrl); err == nil && uri.Scheme == "https" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("[webhook] Failed to callback Kubernetes Enterprise Platform to register the cluster [%v] \n ", err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("[webhook] Failed to ReadAll cluster info [%v],Body: %v \n", err, string(body))
	}
	fmt.Println("[webhook] This Kubernetes cluster registered to Kubernetes Enterprise Platform Successfully \n ")
	return nil
}

func randInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}

// Create long-term token for TenxCloud Enterprise Server in /etc/kubernetes/pki/tokens.csv
func CreateTokenAuthFile(certificatesDir, tokenSecret string) error {
	fmt.Printf("[token-auth] Creating tokens.csv file. tokenSecret [%s]\n", tokenSecret)
	//bts, _:= kubeadmapi.NewBootstrapTokenString(cfg.BootstrapTokens[0].Token.String())
	tokenAuthFilePath := path.Join(certificatesDir, "tokens.csv")
	serialized := []byte(fmt.Sprintf("%s,admin,%s,system:masters\n", tokenSecret, uuid.NewUUID()))
	// DumpReaderToFile create a file with mode 0600
	if err := cmdutil.DumpReaderToFile(bytes.NewReader(serialized), tokenAuthFilePath); err != nil {
		return fmt.Errorf("Failed to save token auth file (%q) [%v]\n", tokenAuthFilePath, err)
	}
	return nil
}
