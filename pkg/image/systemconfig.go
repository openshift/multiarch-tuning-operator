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

package image

import "os"

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
