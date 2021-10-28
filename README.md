# User Guide

## Basic Usage
The following command exports the intermediate Excel file, a.k.a the baseline file.
```Bash
policy_exporter export-intermediate test_resources --config test_resources/example_config.yaml
```

The following command exports the JSON parameter files and MDX files.
Note it generates these files under the directory as specified by `TargetDir`. It will
NOT automatically move these files under corresponding directories under the local Azure LZ
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
OldBaselineExcelFilePath | No | - | The path to the old baseline Excel file, which contains 'Justification', 'Cost Impact' and 'Recommendation' etc.
ExcelFilePath | Yes | - | This is required when performing 'export-final' as it contains parameter values for management groups and subscriptions, which are specified manually.
ManagementGroups | Yes | - | The name of the management groups. It is allowed to provide parameter values per management group.
Subscriptions | Yes | - | The name of subscriptions. For ASC policy parameters, subscription names are used instead of management groups as they apply at subscription level.
TargetDir | No | - | The directory under which the exported files are placed. If not provided then need to be specified through CLI directly.
LocalLandingZoneRepoDir | Yes | - | The path to local directory of the Azure LZ repository.

An example of the configuration file is provided [here](test_resources/example_config.yaml).

In addition to the configurations, the following environment variables must be set for Azure API
authentication, based on the [offical doc](https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authorization#use-environment-based-authentication):

Environment Variable Name | Description
--- | ---
AZURE_TENANT_ID | The tenant ID
AZURE_CLIENT_ID | The client ID 
AZURE_CLIENT_SECRET | The client secret

## Guide for editing the baseline file
The baseline file, which is an Excel file, is mostly used for collecting parameter values.
- If a parameter does not have default value as indicated by the 'Default Values' column, then value must be provided manually if the policy need to be deployed for a management group.
- If all parameters of a policy have default values, then it is possible to indicate that policy will be deployed by simply using **Yes**(case-insensitive) as the cell value for a management group.

A parameter can be of the following types,
- integer: 1, "1", '1' are all treated as integer 1.
- boolean: true, "True" are all treated as boolean true.
- array: <  1, 2 >, \["aa",  "b"\] are all treated as array.
- string: all other values are treated as string.