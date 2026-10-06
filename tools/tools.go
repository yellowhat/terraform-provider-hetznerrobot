// https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-documentation-generation#run-documentation-generation
// https://github.com/hashicorp/terraform-provider-hashicups/blob/main/11-documentation-generation/tools/tools.go

//go:build generate

package tools

import (
	_ "github.com/hashicorp/copywrite"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

// Format examples before they are embedded in docs.
//go:generate terraform fmt -recursive ../examples/

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. -provider-name hetznerrobot
