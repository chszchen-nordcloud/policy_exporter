package exporter

import (
	"context"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
)

func BuildCLI() *cobra.Command {
	var configFile string
	rootCmd := &cobra.Command{
		Use:   "azure-policy-exporter",
		Short: "exports azure policies to various file formats",
		Long: `azure-policy-exporter gathers azure policy information from different sources,
such as Azure API, Excel file and YAML file. For now it supports exporting policies in three formats:
1) JSON files, which are used as parameter file in Azure Landing Zone pipelines. 
2) MDX files, which serves as documentation for the policies.
3) Excel files, which are used to collect user inputs.`,
	}
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "path to the YAML config file")
	subcommands := exportCommands(&configFile)
	for i := range subcommands {
		rootCmd.AddCommand(&subcommands[i])
	}
	return rootCmd
}

func exportCommands(configFile *string) []cobra.Command {
	return []cobra.Command{
		{
			Use:   "export-intermediate [targetDir]",
			Short: "Export intermediate file for collecting user inputs",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				config, err := buildConfig(configFile, args)
				if err != nil {
					return err
				}
				return ExportIntermediateFiles(context.Background(), *config)
			},
		},
		{
			Use:   "export-final [targetDir]",
			Short: "Export final files with all information in place",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				config, err := buildConfig(configFile, args)
				if err != nil {
					return err
				}
				return ExportFinal(context.Background(), *config)
			},
		},
	}
}

func buildConfig(configFile *string, args []string) (*Config, error) {
	b, err := os.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}
	config.TargetDir = args[0]
	return &config, nil
}
