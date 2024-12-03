/*
Copyright 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package systemconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type registryCertTuple struct {
	registry string
	cert     string
}

var (
	dockerCertsDir,
	registriesCertsDir,
	registriesConfPath,
	policyConfPath string
)

func DockerCertsDir() string {
	if dockerCertsDir == "" {
		dockerCertsDir = lookupEnvOr("DOCKER_CERTS_DIR", "/etc/docker/certs.d")
	}
	return dockerCertsDir
}

func RegistryCertsDir() string {
	if registriesCertsDir == "" {
		registriesCertsDir = lookupEnvOr("REGISTRIES_CERTS_DIR", "/etc/containers/registries.d")
	}
	return registriesCertsDir
}

func RegistriesConfPath() string {
	if registriesConfPath == "" {
		registriesConfPath = lookupEnvOr("REGISTRIES_CONF_PATH", "/etc/containers/registries.conf")
	}
	return registriesConfPath
}

func PolicyConfPath() string {
	if policyConfPath == "" {
		policyConfPath = lookupEnvOr("POLICY_CONF_PATH", "/etc/containers/policy.json")
	}
	return policyConfPath
}

func lookupEnvOr(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func (t registryCertTuple) writeToFile() error {
	// create folder if it doesn't exist
	absoluteFolderPath := fmt.Sprintf("%s/%s", DockerCertsDir(), t.getFolderName())
	if _, err := os.Stat(absoluteFolderPath); os.IsNotExist(err) {
		err = os.MkdirAll(absoluteFolderPath, 0700)
		if err != nil {
			return err
		}
	}
	// write cert to file
	absoluteFilePath := fmt.Sprintf("%s/%s/ca.crt", DockerCertsDir(), t.getFolderName())
	f, err := os.Create(filepath.Clean(absoluteFilePath))
	if err != nil {
		return err
	}
	defer close(f)
	_, err = f.WriteString(t.cert)
	if err != nil {
		return err
	}
	return nil
}

func (t registryCertTuple) getFolderName() string {
	// the registry name could report the port number after two dots, e.g. registry.example.com..5000.
	// we need to replace the two dots with a colon to get the correct folder name.
	return strings.Replace(t.registry, "..", ":", 1)
}

func close(f *os.File) {
	err := f.Close()
	if err != nil {
		log.Error(err, "When cosing fd")
	}
}
