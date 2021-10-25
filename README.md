# User Guide

## Basic Usage
The following command exports the intermediate Excel file, which is used for discussion and
to collect user inputs.
```Bash
policy_exporter export-intermediate test_resources --config test_resources/example_config.yaml
```

The following command exports the JSON parameter files and MDX files.
Note it generates these files under the directory as specified by `TargetDir`. It will
not automatically move these files under corresponding directories of the local Azure LZ
repository.
```Bash
policy_exporter export-final test_resources --config test_resources/example_config.yaml
```
## Configurations
The configurations are provided in a YAML file, with the following parameters:

Configuration Name | Required | Default Value | Description 
--- | --- | --- | ---
SubscriptionID | Yes | - | The ID of the subscription under which Azure policies are fetched |
PolicyQueryASCPolicySetName | No | 1f3afdf9-d0c9-4c3d-847f-89da613e70a8 | The ID of the initiative to fetch ASC policy parameters
PolicyQueryManagementGroupName | No | Sandbox | The management group used to fetch builtin policies
OldBaselineExcelFilePath | No | - | The path to the old baseline Excel file. Currently it is needed to reuse justification values. Once a new baseline file is ready this could be replaced.
ExcelFilePath | Yes | - | This is required when exporting the JSON parameter files and MDX files.
YAMLFilePath | No | - | Previously it was in the design to provide similar inputs as currently provided through the Excel file.
ManagementGroups | Yes | - | The name of the management groups. It is allowed to provide parameter values per management group.
Subscriptions | Yes | - | The name of subscriptions. For ASC policy parameters, subscription names are used instead of management groups as they apply at subscription level.
TargetDir | Yes | - | The directory under which the exported files are placed.
LocalLandingZoneRepoDir | Yes | - | The path to local directory of the Azure LZ repository.

An example of the configuration file is provided [here](test_resources/example_config.yaml).

In addition to the configurations, the following environment variables must be set for Azure API
authentication, based on the [offical doc](https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authorization#use-environment-based-authentication):

Environment Variable Name | Description
--- | ---
AZURE_TENANT_ID | The tenant ID
AZURE_CLIENT_ID | The client ID 
AZURE_CLIENT_SECRET | The client secret