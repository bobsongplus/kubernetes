/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package certs

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
	"k8s.io/kubernetes/staging/src/k8s.io/client-go/util/keyutil"
)

var (
	UseEncryption = false
)

func createEncryptionAtRestToken(certsDir string) (tokenB64 string, err error) {
	fileName := filepath.Join(certsDir, kubeadmconstants.EncryptionConfigFileName)
	baseName := KubeadmCertRootCA.Name
	f, osErr := os.Stat(fileName)
	if f != nil && osErr == nil {
		fmt.Printf("[certs] Using existing encryption token and config file %s \n", fileName)
		UseEncryption = true
		return "", nil
	}
	if os.IsNotExist(osErr) {
		_, caKey, err := pkiutil.TryLoadCertAndKeyFromDisk(certsDir, baseName)
		if err != nil {
			return "", errors.Wrapf(err, "failure loading %s certificate authority", baseName)
		}
		encoded, err := keyutil.MarshalPrivateKeyToPEM(caKey)
		if err != nil {
			return "", errors.Wrapf(err, "unable to marshal private key to PEM")
		}
		token := fmt.Sprintf("%x", md5.Sum(encoded))
		tokenB64 = base64.StdEncoding.EncodeToString([]byte(token))
		return tokenB64, nil
	}
	return "", errors.Wrapf(err, "unable to detect config file status")
}

func createEncryptionAtRestConfig(certsDir string, tokenB64 string) error {
	if tokenB64 == "" {
		return nil
	}
	fileName := filepath.Join(certsDir, kubeadmconstants.EncryptionConfigFileName)
	raw := fmt.Sprintf(`kind: EncryptionConfiguration
apiVersion: apiserver.config.k8s.io/v1
resources:
  - resources:
    - secrets
    providers:
    - aescbc:
        keys:
        - name: key1
          secret: %s
    - identity: {}
`, tokenB64)
	err := ioutil.WriteFile(fileName, []byte(raw), os.FileMode(0600))
	if err != nil {
		return err
	}
	fmt.Printf("[certs] Generate Encryption Token and Config file\n")
	UseEncryption = true
	return nil
}

func CreateEncryptionTokenAndConfig(certsDir string) error {
	tokenB64, err := createEncryptionAtRestToken(certsDir)
	if err != nil {
		return err
	}
	return createEncryptionAtRestConfig(certsDir, tokenB64)
}
