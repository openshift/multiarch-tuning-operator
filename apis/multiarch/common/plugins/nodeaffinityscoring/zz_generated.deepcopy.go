//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package nodeaffinityscoring

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeAffinityScoring) DeepCopyInto(out *NodeAffinityScoring) {
	*out = *in
	out.BasePlugin = in.BasePlugin
	if in.Platforms != nil {
		in, out := &in.Platforms, &out.Platforms
		*out = make([]NodeAffinityScoringPlatformTerm, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeAffinityScoring.
func (in *NodeAffinityScoring) DeepCopy() *NodeAffinityScoring {
	if in == nil {
		return nil
	}
	out := new(NodeAffinityScoring)
	in.DeepCopyInto(out)
	return out
}
