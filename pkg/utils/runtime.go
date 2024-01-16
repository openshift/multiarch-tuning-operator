package utils

import "os"

var namespace string
var image string

func init() {
	// Get the namespace from the env variable
	if ns, ok := os.LookupEnv("NAMESPACE"); ok {
		namespace = ns
	} else {
		namespace = "default"
	}
	if img, ok := os.LookupEnv("IMAGE"); ok {
		image = img
	} else {
		image = "quay.io/example/example-operator:latest"
	}
}

// Namespace returns the namespace where the operator's pods are running.
func Namespace() string {
	return namespace
}

// Image returns the image used to run the operator.
func Image() string {
	return image
}
