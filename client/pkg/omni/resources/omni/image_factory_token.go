// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package omni

import (
	"strings"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/resource/protobuf"
	"github.com/cosi-project/runtime/pkg/resource/typed"

	"github.com/siderolabs/omni/client/api/omni/specs"
	"github.com/siderolabs/omni/client/pkg/omni/resources"
)

// NewImageFactoryToken creates a new ImageFactoryToken resource.
//
// The ID is the factory base URL, normalized the same way [NewImageFactoryAuth] normalizes it, so
// that both resources of a single factory share an ID.
func NewImageFactoryToken(id string) *ImageFactoryToken {
	return typed.NewResource[ImageFactoryTokenSpec, ImageFactoryTokenExtension](
		resource.NewMetadata(resources.DefaultNamespace, ImageFactoryTokenType, strings.TrimRight(id, "/"), resource.VersionUndefined),
		protobuf.NewResourceSpec(&specs.ImageFactoryTokenSpec{}),
	)
}

const (
	// ImageFactoryTokenType is the type of ImageFactoryToken resource.
	//
	// tsgen:ImageFactoryTokenType
	ImageFactoryTokenType = resource.Type("ImageFactoryTokens.omni.sidero.dev")
)

// ImageFactoryToken resource holds the machine-to-machine access token Omni presents to an image factory.
type ImageFactoryToken = typed.Resource[ImageFactoryTokenSpec, ImageFactoryTokenExtension]

// ImageFactoryTokenSpec wraps specs.ImageFactoryTokenSpec.
type ImageFactoryTokenSpec = protobuf.ResourceSpec[specs.ImageFactoryTokenSpec, *specs.ImageFactoryTokenSpec]

// ImageFactoryTokenExtension provides auxiliary methods for ImageFactoryToken resource.
type ImageFactoryTokenExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (ImageFactoryTokenExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type:             ImageFactoryTokenType,
		Aliases:          []resource.Type{},
		DefaultNamespace: resources.DefaultNamespace,
		// The token itself is a credential, so it is deliberately not a print column.
		PrintColumns: []meta.PrintColumn{
			{
				Name:     "Expires",
				JSONPath: "{.expiresat}",
			},
		},
		Sensitivity: meta.Sensitive,
	}
}
