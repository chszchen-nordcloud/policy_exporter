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
such as Azure API, Excel file and YAML file. For now it supports exporting policies in two formats:
1) JSON files, which are used as parameter file in Azure Landing Zone pipelines. 
2) MDX files, which serves as documentation for the policies.`,
	}
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "path to the YAML config file")
	rootCmd.AddCommand(cmdExport(&configFile))
	return rootCmd
}

func cmdExport(configFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "export [targetDir]",
		Short: "Export policy to different file formats",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := os.ReadFile(*configFile)
			if err != nil {
				return err
			}

			var config Config
			if err := yaml.Unmarshal([]byte(b), &config); err != nil {
				return err
			}
			config.TargetDir = args[0]

			return SyncAndExport(context.Background(), config)
		},
	}
}
