package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/spf13/cobra"

	"github.com/simongdavies/atfcnab/pkg/template"
)

const (
	bundlecontainerregistry = "cnabquickstartstest.azurecr.io/"
)

var rootCmd = &cobra.Command{
	Use:   "atfcnab",
	Short: "atfcnab generates an ARM template for executing a CNAB package using Azure ACI",
	Long:  `atfcnab generates an ARM template which can be used to execute Duffle in a container using ACI to perform actions on a CNAB Package, which in turn executes the CNAB Actions using the Duffle ACI Driver   `,
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateTemplate()

	},
	SilenceUsage: true,
}

// Execute runs the template generator
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func generateTemplate() error {

	bundle, err := loadBundle("./bundle.json")

	if err != nil {
		fmt.Printf("Failed to load Bundle: %v", err)
		return err
	}

	// TODO need to translate new json schema based parameter format to ARM template format

	generatedTemplate := template.NewTemplate()

	// TODO need to fix this when duffle and porter support bundle push/install from registry
	bundleName, _ := getBundleName(bundle)
	environmentVariable := template.EnvironmentVariable{
		Name:  "CNAB_BUNDLE_NAME",
		Value: bundleName,
	}
	generatedTemplate.SetContainerEnvironmentVariable(environmentVariable)

	for n, p := range bundle.Parameters {
		var metadata template.Metadata
		if p.Metadata != nil && p.Metadata.Description != "" {
			metadata = template.Metadata{
				Description: p.Metadata.Description,
			}
		}

		var allowedValues interface{}
		if p.AllowedValues != nil {
			allowedValues = p.AllowedValues
		}

		var defaultValue interface{}
		if p.Default != nil {
			defaultValue = p.Default
		}

		generatedTemplate.Parameters[n] = template.Parameter{
			Type:          p.DataType,
			AllowedValues: allowedValues,
			DefaultValue:  defaultValue,
			Metadata:      &metadata,
		}
		environmentVariable := template.EnvironmentVariable{
			Name:  strings.ToUpper(n),
			Value: fmt.Sprintf("[parameters('%s')]", n),
		}

		generatedTemplate.SetContainerEnvironmentVariable(environmentVariable)

	}

	for n := range bundle.Credentials {

		var environmentVariable template.EnvironmentVariable

		// TODO update to support description and required attributes once CNAB go is updated

		// Handle TenantId and SubscriptionId as default values from ARM template functions
		if n == "azure_subscription_id" || n == "azure_tenant_id" {
			environmentVariable = template.EnvironmentVariable{
				Name:  strings.ToUpper(n),
				Value: fmt.Sprintf("[subscription().%sId]", strings.TrimSuffix(strings.TrimPrefix(n, "azure_"), "_id")),
			}
		} else {
			generatedTemplate.Parameters[n] = template.Parameter{
				Type: "securestring",
			}
			environmentVariable = template.EnvironmentVariable{
				Name:        strings.ToUpper(n),
				SecureValue: fmt.Sprintf("[parameters('%s')]", n),
			}
		}

		generatedTemplate.SetContainerEnvironmentVariable(environmentVariable)
	}

	res, _ := json.Marshal(generatedTemplate)
	output := string(res)
	fmt.Println(output)

	return nil
}
func getBundleName(bundle *bundle.Bundle) (string, error) {

	for _, i := range bundle.InvocationImages {
		if i.ImageType == "docker" {
			return strings.TrimPrefix(strings.Split(i.Image, ":")[0], bundlecontainerregistry), nil
		}
	}

	return "", fmt.Errorf("Cannot get bundle name from invocationImages: %v", bundle.InvocationImages)
}
func loadBundle(source string) (*bundle.Bundle, error) {
	_, err := os.Stat(source)
	if err == nil {
		jsonFile, _ := os.Open("./bundle.json")
		bundle, err := bundle.ParseReader(jsonFile)
		return &bundle, err
	}
	return nil, err
}
